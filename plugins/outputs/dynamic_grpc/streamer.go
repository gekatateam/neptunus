package dynamicgrpc

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

// This error must never be retried
var ErrEncodingFailed = fmt.Errorf("message encoding failed")

type Streamer struct {
	*core.BaseOutput
	procedure       string
	subscribeFunc   func(rpc string, peer *peer.Peer) subscription
	unsubscribeFunc func(subscription)

	recvMsg protoreflect.MessageDescriptor
	respMsg protoreflect.MessageDescriptor
}

func (s *Streamer) HandleServerStream(_ any, stream grpc.ServerStream) error {
	headers, _ := metadata.FromIncomingContext(stream.Context())
	peer, _ := peer.FromContext(stream.Context())

	m := dynamicpb.NewMessage(s.recvMsg)
	if err := stream.RecvMsg(m); err != nil {
		s.Log.Error("failed to receive initial message from client",
			"error", err,
			"procedure", s.procedure,
			"peer", peer.String(),
		)
		return err
	}

	recv, err := protojson.Marshal(m)
	if err != nil {
		s.Log.Warn("failed to decode initial message from client",
			"error", err,
			"procedure", s.procedure,
			"peer", peer.String(),
		)
		goto DEBUG_INITIAL_MESSAGE_DONE
	}

	s.Log.Debug(fmt.Sprintf("stream initial message received: %v; with headers: %v", string(recv), headers),
		"procedure", s.procedure,
		"peer", peer.String(),
	)
DEBUG_INITIAL_MESSAGE_DONE:

	sub := s.subscribeFunc(s.procedure, peer)
	defer s.unsubscribeFunc(sub)

	for {
		select {
		case <-stream.Context().Done():
			close(sub.result)
			return stream.Context().Err()
		case e, ok := <-sub.events:
			if !ok {
				return nil
			}

			m := dynamicpb.NewMessage(s.respMsg)
			if err := protomap.AnyToMessage(e.Data, m, interceptors.DurationEncoder, interceptors.TimeEncoder); err != nil {
				sub.result <- ErrEncodingFailed
				continue
			}

			if err := stream.SendMsg(m); err != nil {
				s.Log.Error("message sending failed",
					"error", err,
					"procedure", s.procedure,
					"peer", peer.String(),
					elog.EventGroup(e),
				)
				sub.result <- err
				return err
			}

			s.Log.Debug("message sent",
				"procedure", s.procedure,
				"peer", peer.String(),
				elog.EventGroup(e),
			)
			sub.result <- nil
		}
	}
}

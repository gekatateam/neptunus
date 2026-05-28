package dynamicgrpc

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/neptunus/core"
)

type Streamer struct {
	*core.BaseOutput
	procedure       string
	subscribeFunc   func(rpc string, peer *peer.Peer) subscription
	unsubscribeFunc func(subscription)

	recvMsg protoreflect.MessageDescriptor
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
		case m, ok := <-sub.msgs:
			if !ok {
				return nil
			}

			if err := stream.SendMsg(m); err != nil {
				sub.result <- err
				return err
			}

			sub.result <- nil
		}
	}
}

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

type Streamer struct {
	*core.BaseOutput
	Procedure string
	Register func(procedure string, peer *peer.Peer) (events chan *core.Event, result chan error)

	RecvMsg protoreflect.MessageDescriptor
	RespMsg protoreflect.MessageDescriptor
}

func (s *Streamer) HandleServerStream(_ any, stream grpc.ServerStream) error {
	peer, _ := peer.FromContext(stream.Context())
	events, result := s.Register(s.Procedure, peer)
	defer close(result)

	headers, _ := metadata.FromIncomingContext(stream.Context())

	m := dynamicpb.NewMessage(s.RecvMsg)
	err := stream.RecvMsg(m)

	if err != nil {
		s.Log.Error("failed to receive initial message from client",
			"error", err,
			"procedure", s.Procedure,
		)
		return err
	}

	var recv []byte
	if err := protojson.Unmarshal(recv, m); err != nil {
		s.Log.Warn("failed to decode initial message from client",
			"error", err,
			"procedure", s.Procedure,
		)
		goto DEBUG_INITIAL_MESSAGE_DONE
	}

	s.Log.Debug(fmt.Sprintf("stream initial message received: %v; with headers: %v", string(recv), headers),
		"procedure", s.Procedure,
	)
DEBUG_INITIAL_MESSAGE_DONE:

	for e := range events {
		m := dynamicpb.NewMessage(s.RespMsg)
		if err := protomap.AnyToMessage(e.Data, m, interceptors.DurationEncoder, interceptors.TimeEncoder); err != nil {
			s.Log.Error("message encoding failed",
				"error", err,
				"procedure", s.Procedure,
				elog.EventGroup(e),
			)

			result <- err
			continue
		}

		if err := stream.SendMsg(m); err != nil {
			s.Log.Error("message sending failed",
				"error", err,
				"procedure", s.Procedure,
				elog.EventGroup(e),
			)
			result <- err
			return err
		}
	}

	return nil
}

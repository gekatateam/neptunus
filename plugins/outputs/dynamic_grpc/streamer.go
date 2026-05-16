package dynamicgrpc

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"

	"github.com/gekatateam/neptunus/core"
)

type Streamer struct {
	*core.BaseOutput
	Procedure string

	Recv   chan *core.Event
	Result chan error

	RespMsg protoreflect.MessageDescriptor
	RecvMsg protoreflect.MessageDescriptor
}

func (s *Streamer) HandleServerStream(_ any, stream grpc.ServerStream) error {
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

	e, ok := <-s.Recv
	if !ok {
		panic("recv chan closed before stream shutted down")
	}

	m = dynamicpb.NewMessage(s.RespMsg)
	protomap.AnyToMessage(e.Data, m, interceptors.DurationEncoder, interceptors.TimeEncoder)

}

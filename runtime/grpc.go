package runtime

import (
	pubsubpb "github.com/nitrictech/suga/proto/pubsub/v2"
	storagepb "github.com/nitrictech/suga/proto/storage/v2"
)

type GrpcServer struct {
	pubsubpb.UnimplementedPubsubServer
	storagepb.UnimplementedStorageServer
}

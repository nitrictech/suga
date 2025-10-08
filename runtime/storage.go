package runtime

import (
	"context"

	storagepb "github.com/nitrictech/suga/proto/storage/v2"
	"github.com/nitrictech/suga/runtime/storage"
)

func (s *GrpcServer) Read(ctx context.Context, req *storagepb.StorageReadRequest) (*storagepb.StorageReadResponse, error) {
	plugin, err := storage.GetPluginByResourceName(req.BucketName)
	if err != nil {
		return nil, err
	}

	return plugin.Read(ctx, req)
}

func (s *GrpcServer) Write(ctx context.Context, req *storagepb.StorageWriteRequest) (*storagepb.StorageWriteResponse, error) {
	plugin, err := storage.GetPluginByResourceName(req.BucketName)
	if err != nil {
		return nil, err
	}

	return plugin.Write(ctx, req)
}

func (s *GrpcServer) Delete(ctx context.Context, req *storagepb.StorageDeleteRequest) (*storagepb.StorageDeleteResponse, error) {
	plugin, err := storage.GetPluginByResourceName(req.BucketName)
	if err != nil {
		return nil, err
	}

	return plugin.Delete(ctx, req)
}

func (s *GrpcServer) PreSignUrl(ctx context.Context, req *storagepb.StoragePreSignUrlRequest) (*storagepb.StoragePreSignUrlResponse, error) {
	plugin, err := storage.GetPluginByResourceName(req.BucketName)
	if err != nil {
		return nil, err
	}

	return plugin.PreSignUrl(ctx, req)
}

func (s *GrpcServer) ListBlobs(ctx context.Context, req *storagepb.StorageListBlobsRequest) (*storagepb.StorageListBlobsResponse, error) {
	plugin, err := storage.GetPluginByResourceName(req.BucketName)
	if err != nil {
		return nil, err
	}

	return plugin.ListBlobs(ctx, req)
}

func (s *GrpcServer) Exists(ctx context.Context, req *storagepb.StorageExistsRequest) (*storagepb.StorageExistsResponse, error) {
	plugin, err := storage.GetPluginByResourceName(req.BucketName)
	if err != nil {
		return nil, err
	}

	return plugin.Exists(ctx, req)
}

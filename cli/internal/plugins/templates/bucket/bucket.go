package awss3

import (
	"context"

	storagepb "github.com/nitrictech/suga/proto/storage/v2"
	"github.com/nitrictech/suga/runtime/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type plugin struct {
	storagepb.UnimplementedStorageServer
}

func (s *plugin) Read(ctx context.Context, req *storagepb.StorageReadRequest) (*storagepb.StorageReadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *plugin) Write(ctx context.Context, req *storagepb.StorageWriteRequest) (*storagepb.StorageWriteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *plugin) Delete(ctx context.Context, req *storagepb.StorageDeleteRequest) (*storagepb.StorageDeleteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *plugin) PreSignUrl(ctx context.Context, req *storagepb.StoragePreSignUrlRequest) (*storagepb.StoragePreSignUrlResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *plugin) ListBlobs(ctx context.Context, req *storagepb.StorageListBlobsRequest) (*storagepb.StorageListBlobsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *plugin) Exists(ctx context.Context, req *storagepb.StorageExistsRequest) (*storagepb.StorageExistsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method not implemented")
}

func Plugin() (storage.Storage, error) {
	return &plugin{}, nil
}

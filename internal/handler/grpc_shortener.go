package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	pb "github.com/Evlushin/shorturl/proto"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCServer struct {
	pb.UnimplementedShortenerServiceServer
	shortener Shortener
	cfg       config.Config
}

func NewGRPCServer(cfg config.Config, shortener Shortener) *GRPCServer {
	return &GRPCServer{
		shortener: shortener,
		cfg:       cfg,
	}
}

func (s *GRPCServer) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var userID string
	if ok && len(md["user-id"]) > 0 {
		userID = md["user-id"][0]
	} else {
		userID = uuid.New().String()
		if err := grpc.SetHeader(ctx, metadata.Pairs("user-id", userID)); err != nil {
			return nil, err
		}
	}

	resp, err := s.shortener.SetShortener(ctx, &models.SetShortenerRequest{
		URL:    req.GetUrl(),
		UserID: userID,
	})

	isErrConflictURL := errors.Is(err, myerrors.ErrConflictURL)
	if err != nil && !isErrConflictURL {
		if errors.Is(err, myerrors.ErrGetShortenerInvalidRequest) || errors.Is(err, myerrors.ErrValidateShortenerInvalidRequest) {
			logger.Log.Debug("bad request", zap.Error(err))
			return nil, status.Error(codes.InvalidArgument, "bad request")
		}

		logger.Log.Error("failed set shortener", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	fullURL := fmt.Sprintf("%s/%s", s.cfg.BaseAddr, resp.ID)

	return &pb.URLShortenResponse{
		Result: proto.String(fullURL),
	}, nil
}

func (s *GRPCServer) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	resp, err := s.shortener.GetShortener(ctx, &models.GetShortenerRequest{
		ID: req.GetId(),
	})
	if err != nil {
		if errors.Is(err, myerrors.ErrGetShortenerInvalidRequest) || errors.Is(err, myerrors.ErrValidateShortenerInvalidRequest) || errors.Is(err, myerrors.ErrGetShortenerNotFound) {
			logger.Log.Debug("bad request", zap.Error(err))
			return nil, status.Error(codes.InvalidArgument, "bad request")
		}

		if errors.Is(err, myerrors.ErrGone410) {
			logger.Log.Debug("bad request", zap.Error(err))
			return nil, status.Error(codes.NotFound, "item deleted")
		}

		logger.Log.Error("failed to get shortener", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	res := pb.URLExpandResponse{
		Result: proto.String(resp.URL),
	}
	return &res, nil
}

func (s *GRPCServer) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var userID string
	if ok && len(md["user-id"]) > 0 {
		userID = md["user-id"][0]
	} else {
		logger.Log.Debug("user ID not found in config")
		return nil, status.Error(codes.Unauthenticated, "user ID not found in config")
	}

	shorteners, err := s.shortener.GetShortenerUrls(ctx, userID)
	if err != nil {
		if errors.Is(err, myerrors.ErrGetShortenerNotFound) {
			logger.Log.Debug("no content", zap.Error(err))
			return nil, status.Error(codes.NotFound, "no content")
		}

		logger.Log.Error("failed to get shortener", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	resp := make([]*pb.URLData, 0, len(shorteners))
	for _, shortener := range shorteners {
		fullURL := fmt.Sprintf("%s/%s", s.cfg.BaseAddr, shortener.ID)
		resp = append(resp, &pb.URLData{
			ShortUrl:    proto.String(fullURL),
			OriginalUrl: proto.String(shortener.URL),
		})
	}

	return &pb.UserURLsResponse{
		Url: resp,
	}, nil
}

package grpcserver

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	sv_grpc "github.com/biubiu2408/MangaHub/internal/grpc"
	pb "github.com/biubiu2408/MangaHub/internal/grpc/manga"
	"github.com/biubiu2408/MangaHub/internal/manga"
	"github.com/biubiu2408/MangaHub/internal/tcp"
	"github.com/biubiu2408/MangaHub/internal/user"
	"github.com/biubiu2408/MangaHub/package/database"
)

func StartGRPCServer(repo manga.Repository, tcpHub *tcp.Hub, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer()

	// Create user repository for progress updates
	userRepo := user.NewRepository(database.DB)

	pb.RegisterMangaServiceServer(
		grpcServer,
		sv_grpc.NewServer(repo, userRepo, tcpHub),
	)

	log.Printf("📡 gRPC server listening on :%d\n", port)

	// BLOCKING call — caller should run this in a goroutine
	return grpcServer.Serve(lis)
}

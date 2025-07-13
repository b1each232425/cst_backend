package cmn

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"io"
	"net"
	"regexp"
	"time"
	pb "w2w.io/w2wproto"
)

type grpcServer struct {
	pb.UnimplementedW2WServer
	serveListener net.Listener
	conn          *grpc.ClientConn
	cln           pb.W2WClient
}

type grpcClient struct {
	pb.UnimplementedW2WServer
	serveListener net.Listener
	conn          *grpc.ClientConn
	cln           pb.W2WClient
}

var server grpcServer
var client grpcClient

func (s *grpcServer) Do(task *pb.Task, stream pb.W2W_DoServer) (err error) {
	//func (s *grpcServe) Do(ctx context.Context, task *pb.Task) (reply *pb.Reply, err error) {

	fmt.Printf("action: %s\ndata: %s\n", task.Name, string(task.GetData()))
	msg := "success"
	reply := &pb.Reply{
		Status: 0,
		Msg:    &msg,
		Data:   []byte(`{"name":"老张四子","age":33.2,"gender":"男","weight":88.5592}`),
	}
	err = stream.Send(reply)
	if err != nil {
		z.Error(err.Error())
	}
	return
}

func grpcClnInit() {
	var err error
	// grpc.Dial doesn't do a REAL dial, but construct a context only.
	//   so grpc.Dial should always success.
	client.conn, err = grpc.Dial(fmt.Sprintf("%s:%d", GRPCAddr, GRPCPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(256*1024*1024),
			grpc.MaxCallRecvMsgSize(256*1024*1024)),
	)
	if err != nil {
		z.Error(err.Error())
		return
	}

	client.cln = pb.NewW2WClient(client.conn)
}

func GrpcDo(task *pb.Task, md metadata.MD) (reply *pb.Reply, err error) {
	if client.conn == nil {
		grpcClnInit()
	}

	repoIDs := md.Get("repo-id")
	if len(repoIDs) == 0 || repoIDs[0] == "" {
		z.Error("please set repo-id in metadata")
		return
	}

	task.RepoId = repoIDs[0]

	ctx, cancel := context.WithTimeout(context.Background(), 60*60*time.Second)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, md)

	//z.Info("data: " + string(task.Data))
	// pb.NewW2WClient.Do() calling does an REAL dial.
	var doClient pb.W2W_DoClient
	doClient, err = client.cln.Do(ctx, task)

	//code = Unavailable
	if err != nil {
		target := fmt.Sprintf("%s:%d", GRPCAddr, GRPCPort)
		re := regexp.MustCompile("(?i)code *= *Unavailable")
		if re.MatchString(err.Error()) {
			err = fmt.Errorf("仓库服务%s故障(code=unavailable, 可连接但工作不正常), 请稍后再试", target)
			grpcClnInit()
		}

		re = regexp.MustCompile("(?i)Connection refused")
		if re.MatchString(err.Error()) {
			err = fmt.Errorf("仓库服务%s应用未启动(connection refused), 请稍后再试", target)
			grpcClnInit()
		}

		z.Error(err.Error())
		return
	}

	for {
		reply, err = doClient.Recv()
		if err == io.EOF {
			err = fmt.Errorf("grpc server terminated serice")
			z.Error(err.Error())
			break
		}

		if err != nil {
			z.Error(err.Error())
			break
		}

		// it's progress status response
		if reply.Status == 100 {
			// echo to front end with websocket
			z.Info(*reply.Msg)
			continue
		}

		break
	}
	return
}

// GRPCServe start grpc server
func GRPCServe() {
	var err error
	server.serveListener, err = net.Listen("tcp", fmt.Sprintf(":%d", GRPCPort))
	if err != nil {
		z.Error("grpc listen failed with " + err.Error())
		return
	}

	s := grpc.NewServer()
	pb.RegisterW2WServer(s, &server)
	z.Info(fmt.Sprintf("server listening at %v",
		server.serveListener.Addr()))

	err = s.Serve(server.serveListener)
	if err != nil {
		z.Error("failed to listen: " + err.Error())
		return
	}
}

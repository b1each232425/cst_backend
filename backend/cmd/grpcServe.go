package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc/metadata"
	"log"
	"w2w.io/cmn"
	pb "w2w.io/w2wproto"
)

// grpcServeCmd represents the grpcServe command
var grpcServeCmd = &cobra.Command{
	Use:   "grpcServe",
	Short: "grpc service",
	Long:  `w2w.io grp service.`,
	Run: func(cmd *cobra.Command, args []string) {

		mode := viper.GetString("mode")
		if mode == "c" {
			cln()
			return
		}

		if mode == "s" {
			srv()
			return
		}

		fmt.Println("unknown mode: " + mode)
		log.Fatal("unknown mode: " + mode)

	},
}

func init() {
	rootCmd.AddCommand(grpcServeCmd)

	grpcServeCmd.Flags().StringP("mode", "m", "s", "s: act as grp server, c: act as grpc client")
	err := viper.BindPFlag("mode", grpcServeCmd.Flags().Lookup("mode"))
	if err != nil {
		log.Fatal(err.Error())
	}

	grpcServeCmd.Flags().Int32P("port", "p", 50051, "server/client [listen on]/[connect to] port")
	err = viper.BindPFlag("port", grpcServeCmd.Flags().Lookup("port"))
	if err != nil {
		log.Fatal(err.Error())
	}

	grpcServeCmd.Flags().StringP("target", "t", "localhost", "server address")
	err = viper.BindPFlag("target", grpcServeCmd.Flags().Lookup("target"))
	if err != nil {
		log.Fatal(err.Error())
	}
}

func cln() {
	md := metadata.Pairs("repoID", "localhost/git/kzz/gittravel")
	reply, err := cmn.GrpcDo(&pb.Task{Name: "repo"}, md)
	if err != nil {
		z.Error(err.Error())
		return
	}
	z.Info(*reply.Msg)
}

func srv() {
	cmn.GRPCServe()
}

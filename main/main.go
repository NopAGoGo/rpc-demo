package main

import (
	"context"
	"github.com/megaredfan/rpc-demo/client"
	"github.com/megaredfan/rpc-demo/codec"
	"github.com/megaredfan/rpc-demo/registry/memory"
	"github.com/megaredfan/rpc-demo/server"
	"github.com/megaredfan/rpc-demo/service"
	"github.com/megaredfan/rpc-demo/share/ratelimit"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"log"
	"math/rand"
	"strconv"
	"time"
)

const callTimes = 1

var s1, s2, s3 server.RPCServer

func main() {
	opentracing.SetGlobalTracer(mocktracer.New())
	StartServer()
	time.Sleep(2e9)
	start := time.Now()
	for i := 0; i < callTimes; i++ {
		MakeCall(codec.MessagePack)
	}
	cost := time.Now().Sub(start)
	log.Printf("cost:%s", cost)

	start = time.Now()
	for i := 0; i < callTimes; i++ {
		MakeCall(codec.GOB)
	}
	cost = time.Now().Sub(start)
	log.Printf("cost:%s", cost)

	StopServer()
}

func StopServer() {
	s1.Close()
	s2.Close()
	s3.Close()
}

//var Registry = zookeeper.NewZookeeperRegistry("my-app", "/mns/sankuai/service",
//	[]string{"127.0.0.1:2181"}, 1e10, nil)
var Registry = memory.NewInMemoryRegistry()

func StartServer() {
	go func() {
		serverOpt := server.DefaultOption
		serverOpt.RegisterOption.AppKey = "my-app"
		serverOpt.Registry = Registry
		serverOpt.Tags = map[string]string{"status": "stopped"}

		s1 = server.NewRPCServer(serverOpt)
		err := s1.Register(service.Arith{})
		if err != nil {
			log.Println("err!!!" + err.Error())
		}
		port := 8880
		s1.Serve("tcp", ":"+strconv.Itoa(port), nil)
	}()
	go func() {
		serverOpt := server.DefaultOption
		serverOpt.RegisterOption.AppKey = "my-app"
		serverOpt.Registry = Registry
		serverOpt.Tags = map[string]string{"status": "starting"}

		s2 = server.NewRPCServer(serverOpt)
		err := s2.Register(service.Arith{})
		if err != nil {
			log.Println("err!!!" + err.Error())
		}
		port := 8881
		s2.Serve("tcp", ":"+strconv.Itoa(port), nil)
	}()
	go func() {
		serverOpt := server.DefaultOption
		serverOpt.RegisterOption.AppKey = "my-app"
		serverOpt.Registry = Registry
		serverOpt.Tags = map[string]string{"status": "alive"}

		s3 = server.NewRPCServer(serverOpt)
		err := s3.Register(service.Arith{})
		if err != nil {
			log.Println("err!!!" + err.Error())
		}
		port := 8882
		s3.Serve("tcp", ":"+strconv.Itoa(port), nil)
	}()
}

func MakeCall(t codec.SerializeType) {
	op := &client.DefaultSGOption
	op.AppKey = "my-app"
	op.SerializeType = t
	op.RequestTimeout = time.Millisecond * 100
	op.DialTimeout = time.Millisecond * 100
	op.FailMode = client.FailRetry
	op.Retries = 3

	op.Heartbeat = true
	op.HeartbeatInterval = time.Second * 10
	op.HeartbeatDegradeThreshold = 10
	op.Tagged = true
	op.Tags = map[string]string{"status": "alive"}
	op.Wrappers = append(op.Wrappers, &client.RateLimitInterceptor{Limit: &ratelimit.DefaultRateLimiter{Num: 1}})

	//r := registry.NewPeer2PeerRegistry()
	//r.Register(registry.RegisterOption{}, registry.Provider{ProviderKey: "tcp@:8880", Network: "tcp", Addr: ":8880"})
	op.Registry = Registry

	c := client.NewSGClient(*op)

	args := service.Args{A: rand.Intn(200), B: rand.Intn(100)}
	reply := &service.Reply{}
	ctx := context.Background()
	err := c.Call(ctx, "Arith.Add", args, reply)
	if err != nil {
		log.Println("err!!!" + err.Error())
	} else if reply.C != args.A+args.B {
		log.Printf("%d + %d != %d", args.A, args.B, reply.C)
	}

	args = service.Args{A: rand.Intn(200), B: rand.Intn(100)}
	reply = &service.Reply{}
	ctx = context.Background()
	err = c.Call(ctx, "Arith.Minus", args, reply)
	if err != nil {
		log.Println("err!!!" + err.Error())
	} else if reply.C != args.A-args.B {
		log.Printf("%d - %d != %d", args.A, args.B, reply.C)
	}

	args = service.Args{A: rand.Intn(200), B: rand.Intn(100)}
	reply = &service.Reply{}
	ctx = context.Background()
	err = c.Call(ctx, "Arith.Mul", args, reply)
	if err != nil {
		log.Println("err!!!" + err.Error())
	} else if reply.C != args.A*args.B {
		log.Printf("%d * %d != %d", args.A, args.B, reply.C)
	}

	args = service.Args{A: rand.Intn(200), B: rand.Intn(100)}
	reply = &service.Reply{}
	ctx = context.Background()
	err = c.Call(ctx, "Arith.Divide", args, reply)
	if args.B == 0 && err == nil {
		log.Println("err!!! didn't return errror!")
	} else if err != nil && err.Error() == "divided by 0" {
		log.Println(err.Error())
	} else if err != nil {
		log.Println("err!!!" + err.Error())
	} else if reply.C != args.A/args.B {
		log.Printf("%d / %d != %d", args.A, args.B, reply.C)
	}
}

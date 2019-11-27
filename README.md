# easymicro
- 一个简单易用、好理解的微服务治理框架，实现上参照了net/rpc、rpcx、grpc等诸多优秀的rpc框架，底层数据传输协议使用rpcx协议格式，框架封装则主要在net/rpc库的基础上，增加超时访问、心跳保活、断开重连、元数据传递、服务发现、熔断、限流以及指标统计和链路追踪等特性；服务端支持网关模式，可以处理本服务的http访问请求；同时还提供了一个通用的网关实现，可以用于快速搭建自己的微服务集群。

- 依赖管理

  - go mod

- 快速上手

  - 直接运行examples中的例子即可

- 名词解释

  - server ： rpc服务端的一个实例
  - client：  rpc客户端的一个实例
  - rpcclient： rpc客户端到服务端到一个连接
  - 网关代理：每个服务可以启用网关代理模式，自身支持http到rpc的转换
  - 独立网关：集成服务发现组件，提供所有服务的http协议到rpc协议的转换

- rpcclient

  - rpc客户端到服务端到一个连接

  - 当创建一个rpcclient时，会开启两个goroutine

    ```go
    	go client.input()  //用于接收服务端发送到数据
    	go client.keepalive()  //用于连接保活
    ```

  - 保活设计

    - 根据配置的时间间隔，发送心跳包到对端，当本地有数据发送时，则推迟心跳包到发送，如果心跳包发送失败（会尝试重发几次），则根据当前连接的状态做相应的处理。
    - 当rpcclient关闭时，会退出keepalive goroutine。

- 元数据传递

  - 客户端发送和获取

    ```go
    ctx := metadata.NewClientMdContext(context.Background(), map[string]string{
    		"testmd":  "md",
    		"testMd2": "md2",
    	})
    
    	mdFromServer := map[string]string{}
    	err = client.Call(ctx, "Mul", args, reply, easyclient.GetMetadataFromServer(&mdFromServer))
    	if err != nil {
    		log.Error(err)
    	}
    ```

  - 服务端发送和获取

    ```go
    	md, b := metadata.FromClientMdContext(ctx)
    	if b {
    		log.Infof("get md %+v from context", md)
    	}
    	reply.C = args.A * args.B
    	server.SendMetaData(ctx, map[string]string{
    		"server": "hi this is server md",
    	})
    ```

- 服务发现

  - 目前仅支持etcd

  - 客户端

    ```go
    	cli, err := client.NewDiscoveryClient("Arith", dis.NewEtcdDiscoveryMaster([]string{
    		"http://127.0.0.1:22379",
    	}, "Arith"))
    ```

    

  - 服务端

    ```go
    	s := server.NewServer(server.SetEtcdDiscovery([]string{
    		"http://127.0.0.1:22379",
    	}, "127.0.0.1:8972"))
    
    	s.RegisterName("Arith", new(Arith))
    	s.Serve("tcp", ":8972")
    ```

- 网关代理模式

  - 服务端

    ```go
    	s := server.NewServer(server.SetGateWayMode(":8888"))
    	s.RegisterName("Arith", new(Arith))
    	s.Serve("tcp", ":8972")
    ```

  - TODO 后续会加上网关代理的服务发现

- 独立网关

  - 根据给定的服务发现地址，路由所有的服务

  ```go
  	gw := gateway.NewGateway(":8887", []string{"http://127.0.0.1:22379"})
  	gw.StartGateway()
  ```

- 性能

  - 在请求处理上，采用了goroutine池的方式，没有使用per request per goroutine的模式

  - 主要跟grpc对比了一下，1000个并发，5000000条数据，每条消息大小581B,性能差不多是grpc的两倍多

  - grpc

    ```
    2019/11/27 12:28:16 concurrency: 1000
    requests per client: 5000
    
    2019/11/27 12:28:16 message size: 581 bytes
    
    2019/11/27 12:33:11 took 294315 ms for 5000000 requests
    2019/11/27 12:33:15 sent     requests    : 5000000
    2019/11/27 12:33:15 received requests    : 5000000
    2019/11/27 12:33:15 received requests_OK : 5000000
    2019/11/27 12:33:15 throughput  (TPS)    : 16988
    2019/11/27 12:33:15 mean: 58657349 ns, median: 55538530 ns, max: 7140185427 ns, min: 282498 ns, p99: 164440052 ns
    2019/11/27 12:33:15 mean: 58 ms, median: 55 ms, max: 7140 ms, min: 0 ms, p99: 164 ms
    
    ```

    

  - easymicro

    ```
    2019-11-27 13:40:19.722522 I | took 124799 ms for 5000000 requests
    2019/11/27 13:40:19 rpcclient.go:403: INFO : client input goroutine exit
    2019/11/27 13:40:19 rpcclient.go:308: INFO : client keepalive goroutine exit
    2019-11-27 13:40:23.597393 I | sent     requests    : 5000000
    2019-11-27 13:40:23.597422 I | received requests    : 5000000
    2019-11-27 13:40:23.597429 I | received requests_OK : 5000000
    2019-11-27 13:40:23.597435 I | throughput  (TPS)    : 40064
    2019-11-27 13:40:23.597466 I | mean: 24768440 ns, median: 22327754 ns, max: 3023957318 ns, min: 207311 ns, p99: 65509680 ns
    2019-11-27 13:40:23.597475 I | mean: 24 ms, median: 22 ms, max: 3023 ms, min: 0 ms, p99: 65 ms
    ```

    
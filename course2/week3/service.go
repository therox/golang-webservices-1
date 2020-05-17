package main

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

// Тут мы запускаем наш сервер, передаем ему контекст, адрес, на котором он будет слушать, а так же какую-то строку с
// параметрами разграничения прав доступа
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	// ACLData string = `{
	//	"logger":    ["/main.Admin/Logging"],
	//	"stat":      ["/main.Admin/Statistics"],
	//	"biz_user":  ["/main.Biz/Check", "/main.Biz/Add"],
	//	"biz_admin": ["/main.Biz/*"]
	//}`
	// Распаковываем ACL
	fmt.Println("Распаковываем права")
	var aclData map[string][]string
	err := json.Unmarshal([]byte(ACLData), &aclData)
	if err != nil {
		return err
	}
	fmt.Println("Запускаем сервер")
	srv := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor))

	lis, err := net.Listen("tcp", listenAddr)
	log.Fatal(srv.Serve(lis))
	fmt.Println("Запустили слушать")
	return nil
}

// Пишем интерсептор, который перед запуском Test проверяет ACL
func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	fmt.Println(info.FullMethod)
	h, err := handler(ctx, req)
	if err != nil {
		return nil, err
	}
	return h, nil
}

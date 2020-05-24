package main

import (
	"context"
	"encoding/json"
	fmt "fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

type MyServer struct {
	logs chan *Event
}

//type AdminServer interface {
//	Logging(*Nothing, Admin_LoggingServer) error
//	Statistics(*StatInterval, Admin_StatisticsServer) error
//}

func NewServer() *MyServer {
	return &MyServer{
		logs: make(chan *Event, 10),
	}
}

func (m *MyServer) Logging(n *Nothing, als Admin_LoggingServer) (err error) {
	for {
		select {
		case ev := <-m.logs:
			err = als.Send(ev)
			fmt.Println("struct sended to logging client", ev)
		}
		if err != nil {
			fmt.Println("service closed", err)
			//m.IsLogging = false

			return

		}

	}

	return nil
}

func (m *MyServer) Statistics(si *StatInterval, ass Admin_StatisticsServer) error {
	return nil
}

func (m *MyServer) Check(ctx context.Context, n *Nothing) (*Nothing, error) {
	return n, nil
}

func (m *MyServer) Add(ctx context.Context, n *Nothing) (*Nothing, error) {
	return n, nil
}

func (m *MyServer) Test(ctx context.Context, n *Nothing) (*Nothing, error) {
	return n, nil
}

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
	var aclData map[string][]string
	err := json.Unmarshal([]byte(ACLData), &aclData)
	if err != nil {
		return err
	}
	s := NewServer()
	srv := grpc.NewServer(grpc.UnaryInterceptor(s.authInterceptor(aclData)), grpc.StreamInterceptor(s.authStreamInterceptor(aclData)))

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Ошибка инициализации порта: %s", err)
	}
	RegisterBizServer(srv, s)
	RegisterAdminServer(srv, s)
	//fmt.Println("Запускаем сервер")

	go runGrpcServer(ctx, srv, lis)

	return nil
}

// Пишем интерсептор, который перед запуском Test проверяет ACL
func (s *MyServer) authInterceptor(acl map[string][]string) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		var ev *Event
		if ev, err = s.checkAuth(ctx, acl, info.FullMethod); err != nil {
			return
		}
		//fmt.Println("Method: ", info.FullMethod)
		//fmt.Println("Пропустили")
		h, err := handler(ctx, req)
		if err != nil {
			return nil, err
		}
		s.logs <- ev
		return h, nil
	}
}

func (s *MyServer) authStreamInterceptor(aclData map[string][]string) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ev, err := s.checkAuth(ss.Context(), aclData, info.FullMethod)
		if err != nil {
			return
		}
		s.logs <- ev
		err = handler(srv, ss)
		return
	}
}

func (s *MyServer) checkAuth(ctx context.Context, aclData map[string][]string, method string) (*Event, error) {
	var ev *Event
	var err error
	//fmt.Println("Проверяем аутентификацию...")
	md, _ := metadata.FromIncomingContext(ctx)

	c := md.Get("consumer")
	if len(c) == 0 {
		err = grpc.Errorf(codes.Unauthenticated, "")
		return ev, err
	}
	//fmt.Println("К нам пришел ", c[0], "с  методом ", info.FullMethod)
	isFound := false
	for k := range aclData {
		if k == c[0] {
			isFound = true
			break
		}
	}
	if !isFound {
		err = grpc.Errorf(codes.Unauthenticated, "")
		return ev, err
	}
	//Ищем подходящий метод
	isFound = false
	for _, v := range aclData[c[0]] {
		if strings.HasPrefix(method, strings.Split(v, "*")[0]) {
			isFound = true
			break
		}
	}
	if !isFound {
		err = grpc.Errorf(codes.Unauthenticated, "")
		return ev, err
	}
	return &Event{
		Timestamp:            time.Now().Unix(),
		Consumer:             c[0],
		Method:               method,
		Host:                 md[":authority"][0],
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}, nil
}

func runGrpcServer(ctx context.Context, srv *grpc.Server, lis net.Listener) {
	//fmt.Printf("Мы получили контекст: %+v\n", ctx)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	//fmt.Println("Запускаем сервер")
	go srv.Serve(lis)
Loop:
	for {
		select {
		case <-ctx.Done():
			//fmt.Println("Получили команду на завершение работы сервера")
			srv.GracefulStop()
			wg.Done()
			break Loop
		default:
		}
	}
	//fmt.Println("Ждем, когда отпустит")
	wg.Wait()
	//fmt.Println("Отпустило")
}

package serve

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Dcarbon/arch-proto/pb"
	"github.com/Dcarbon/go-shared/gutils"
	"github.com/Dcarbon/go-shared/libs/container"
	"github.com/Dcarbon/go-shared/libs/utils"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

var swaggerHost = utils.StringEnv("SWAGGER_HOST", "dev01.viet-tin.com")

type RegisterServiceFn[T any] func(context.Context, *runtime.ServeMux, *T) error

type Serve struct {
	*runtime.ServeMux
	swaggerDoc string // Swagger doc content
	clients    *container.SafeMap[string, *grpc.ClientConn]
}

func NewServeMux(swgDocPath string) (*Serve, error) {
	var mux = &Serve{
		ServeMux: runtime.NewServeMux(runtime.WithMarshalerOption(
			runtime.MIMEWildcard, &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					EmitUnpopulated: false,
					UseEnumNumbers:  true,
				},
			}),
		),
		clients: container.NewSafeMap[string, *grpc.ClientConn](),
	}
	mux.HandlePath(http.MethodGet, "/api/v1.1/iot/geojson", mux.GetGeoJson2) //mux.GetGeoJson
	mux.HandlePath(http.MethodGet, "/api/v1.1/dcarbon.json", mux.GetSwagger)
	mux.HandlePath(http.MethodPost, "/api/v1.1/iot-op/version", mux.handleFileUpload)
	mux.Register(
		gutils.ISVIotInfo,
		utils.StringEnv(gutils.ISVIotInfo, "localhost:4002"),
		pb.RegisterIotServiceHandler,
	)
	mux.Register(
		gutils.ISVIotMapListener,
		utils.StringEnv(gutils.ISVIotMapListener, "localhost:4010"),
		pb.RegisterIOTMapListenerServiceHandler,
	)
	mux.Register(
		gutils.ISVIotOp,
		utils.StringEnv(gutils.ISVIotOp, "localhost:4003"),
		pb.RegisterIotOpServiceHandler,
	)

	mux.Register(
		gutils.ISVSensorInfo,
		utils.StringEnv(gutils.ISVSensorInfo, "localhost:4030"),
		pb.RegisterSensorServiceHandler,
	)

	mux.Register(
		gutils.ISVISM,
		utils.StringEnv(gutils.ISVISM, "localhost:4031"),
		pb.RegisterISMServiceHandler,
	)

	mux.Register(
		gutils.ISVIotMapListener,
		utils.StringEnv(gutils.ISVIotMapListener, "localhost:4010"),
		pb.RegisterIOTMapListenerServiceHandler,
	)
	mux.Register(
		gutils.ISVProjects,
		utils.StringEnv(gutils.ISVProjects, "localhost:4012"),
		pb.RegisterProjectServiceHandler,
	)
	mux.Register(
		gutils.ISFinance,
		utils.StringEnv(gutils.ISFinance, "localhost:4199"),
		pb.RegisterFinanceServiceHandler,
	)
	mux.Register(
		gutils.ISNotification,
		utils.StringEnv(gutils.ISNotification, "localhost:4099"),
		pb.RegisterNotificationServiceHandler,
	)
	if swgDocPath != "" {
		raw, err := os.ReadFile(swgDocPath)
		if nil != err {
			return nil, err
		}
		mux.swaggerDoc = fmt.Sprintf(`{"host": "%s",`, swaggerHost) + string(raw[0:])
	}

	return mux, nil
}

func (s *Serve) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Path: ", r.URL.Path)

	s.ServeMux.ServeHTTP(w, r)
}

func (s *Serve) Start(port int) error {

	withCors := cors.New(cors.Options{
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"ACCEPT", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler(s.ServeMux)

	gwServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: withCors,
	}

	log.Printf("Serving gRPC-Gateway on http://0.0.0.0:%d\n", port)
	return gwServer.ListenAndServe()
}

func (s *Serve) Register(sname, host string, fn RegisterServiceFn[grpc.ClientConn]) {
	cc, err := gutils.GetCCTimeout(host, 5*time.Second)
	utils.PanicError("Connect to service "+sname, err)

	err = fn(context.TODO(), s.ServeMux, cc)
	utils.PanicError("Register service "+sname, err)

	s.clients.Set(sname, cc)
	log.Printf("Register serice %s [%s] success\n", sname, host)
}

func (s *Serve) GetSwagger(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(s.swaggerDoc))
}

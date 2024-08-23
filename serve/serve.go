package serve

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Dcarbon/arch-proto/pb"
	"github.com/Dcarbon/go-shared/gutils"
	"github.com/Dcarbon/go-shared/libs/aidh"
	"github.com/Dcarbon/go-shared/libs/container"
	"github.com/Dcarbon/go-shared/libs/utils"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

var swaggerHost = utils.StringEnv("SWAGGER_HOST", "dev01.viet-tin.com")

const jwt = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4MDEzNjczNDEsIkF1dGgiOnsiSWQiOjEsIlJvbGUiOiIiLCJGaXJzdE5hbWUiOiIiLCJMYXN0TmFtZSI6IiIsIlVzZXJuYW1lIjoiVGVzdCJ9fQ.RazlBAfn_nmt47GMSUHE3TXq2f4_mR7KvsXodeQ7Tgo"

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
	mux.HandlePath(http.MethodPost, "/api/v1.1/project/upload", mux.handleFileUpload)
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
	// mux.Register(
	// 	gutils.ISVAUTH,
	// 	utils.StringEnv(gutils.ISVAUTH, "localhost:4005"),
	// 	pb.RegisterAuthServiceHandler,
	// )
	// mux.Register(
	// 	gutils.ISVUser,
	// 	utils.StringEnv(gutils.ISVUser, "localhost:4006"),
	// 	pb.RegisterUserInfoServiceHandler,
	// )
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

		// mux.swaggerDoc = string(raw)
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

func (s *Serve) GetGeoJson(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	cc, ok := s.clients.Get(gutils.ISVIotInfo)
	if !ok {
		w.WriteHeader(400)
		aidh.SendJSON(w, 500, gutils.ErrServiceNotAvailable("IotInfo"))
		return
	}

	iotService := pb.NewIotServiceClient(cc)
	data, err := iotService.GetIotPositions(context.TODO(), &pb.RIotGetList{})
	if nil != err {
		w.WriteHeader(400)
		aidh.SendJSON(w, 500, err)
		return
	}

	var featureCollection = geojson.NewFeatureCollection()
	for _, loc := range data.Data {
		var feature = geojson.NewFeature(&orb.Point{loc.Position.Longitude, loc.Position.Latitude})
		feature.Properties = make(geojson.Properties)
		feature.Properties["id"] = loc.Id
		featureCollection.Append(feature)
	}
	aidh.SendJSON(w, 200, featureCollection)
}

func (s *Serve) GetGeoJson2(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	cc, ok := s.clients.Get(gutils.ISVIotMapListener)
	if !ok {
		w.WriteHeader(400)
		aidh.SendJSON(w, 500, gutils.ErrServiceNotAvailable("IotMapListener"))
		return
	}

	iotService := pb.NewIOTMapListenerServiceClient(cc)
	data, err := iotService.GetIotMapListenerPositions(context.TODO(), &pb.RIotMapGetList{})
	if nil != err {
		w.WriteHeader(400)
		aidh.SendJSON(w, 500, err.Error())
		return
	}

	var featureCollection = geojson.NewFeatureCollection()
	for _, loc := range data.Data {
		var feature = geojson.NewFeature(&orb.Point{loc.Position.Longitude, loc.Position.Latitude})
		feature.Properties = make(geojson.Properties)
		feature.Properties["id"] = loc.Id
		featureCollection.Append(feature)
	}
	aidh.SendJSON(w, 200, featureCollection)
}

func (s *Serve) GetSwagger(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	// aidh.SendJSON(w, 200, s.swaggerDoc)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(s.swaggerDoc))
}

func (s *Serve) handleFileUpload(w http.ResponseWriter, r *http.Request, params map[string]string) {
	// Retrieve the ID from the form
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		aidh.SendJSON(w, http.StatusBadRequest, fmt.Sprintf("invalid id: %s", err.Error()))
		return
	}
	// Retrieve the file from the form
	f, header, err := r.FormFile("file")
	if err != nil {
		aidh.SendJSON(w, http.StatusBadRequest, fmt.Sprintf("failed to get file 'file': %s", err.Error()))
		return
	}
	defer f.Close()
	// Create a temporary file to store the uploaded file
	// tempFile, err := os.CreateTemp("", fmt.Sprintf("upload-*.%s", strings.Split(header.Header.Get("Content-Type"), "/")[1]))
	// if err != nil {
	// 	aidh.SendJSON(w, http.StatusInternalServerError, fmt.Sprintf("failed to create temp file: %s", err.Error()))
	// 	return
	// }
	tempFile, err := os.Create(header.Filename)
	if err != nil {
		aidh.SendJSON(w, http.StatusInternalServerError, fmt.Sprintf("failed to create temp file: %s", err.Error()))
		return
	}
	defer os.Remove(header.Filename)
	defer tempFile.Close()
	// Copy the uploaded file to the temporary file
	if _, err := io.Copy(tempFile, f); err != nil {
		aidh.SendJSON(w, http.StatusInternalServerError, fmt.Sprintf("failed to copy file content: %s", err.Error()))
		return
	}
	// Get the project service client
	cc, ok := s.clients.Get(gutils.ISVProjects)
	if !ok {
		aidh.SendJSON(w, http.StatusInternalServerError, gutils.ErrServiceNotAvailable("ProjectServer"))
		return
	}
	prjService := pb.NewProjectServiceClient(cc)
	// Add the image to the project
	image, err := prjService.AddImage(context.TODO(), &pb.RPAddImage{
		ProjectId: int64(id),
		Image:     "../gateway/" + tempFile.Name(),
	})

	if err != nil {
		aidh.SendJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	// Send a successful response
	aidh.SendJSON(w, http.StatusOK, image)
}

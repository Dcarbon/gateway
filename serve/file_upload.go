package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/Dcarbon/arch-proto/pb"
	"github.com/Dcarbon/go-shared/gutils"
	"github.com/Dcarbon/go-shared/libs/aidh"
	"github.com/Dcarbon/go-shared/libs/utils"
	"github.com/go-resty/resty/v2"
)

var Host = utils.StringEnv("S3_UPLOAD_URL", "localhost")
var Authorization = utils.StringEnv("S3_UPLOAD_AUTHORIZATION", "localhost")

type UploadResponse struct {
	RequestId  string `json:"request_id"`
	StatusCode int    `json:"statusCode"`
	Data       Data   `json:"data"`
}
type Data struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
}

func (s *Serve) handleFileUpload(w http.ResponseWriter, r *http.Request, params map[string]string) {
	//Retrieve the ID from the form

	iotType, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		aidh.SendJSON(w, http.StatusBadRequest, fmt.Sprintf("invalid id: %s", err.Error()))
		return
	}
	version := r.FormValue("version")
	//Retrieve the file from the form
	file, header, err := r.FormFile("file")
	if err != nil {
		aidh.SendJSON(w, http.StatusBadRequest, fmt.Sprintf("failed to get file 'file': %s", err.Error()))
		return
	}
	response, err := MakeRequest(header.Filename, iotType, version, file)
	if err != nil {
		aidh.SendJSON(w, http.StatusBadRequest, err)
		return
	}
	relatePath := strings.SplitN(response.Data.RelativePath, "s3://", 2)
	if err := s.SaveVersion(&pb.RIotSetVersion{IotType: int32(iotType), Version: version, Path: relatePath[1]}); err != nil {
		aidh.SendJSON(w, http.StatusBadRequest, err)
		return
	}
	aidh.SendJSON(w, http.StatusOK, "OK")
}
func MakeRequest(fileName string, iotType int, version string, file multipart.File) (*UploadResponse, error) {
	defer file.Close()
	client := resty.New()
	// Make the PATCH request
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+Authorization).
		SetFormData(map[string]string{
			"secure": "public",
			"key":    fmt.Sprintf("static/iot/ota/%d/%s", iotType, version),
		}).
		SetFileReader("file", fileName, file).
		Patch(Host)
	if err != nil {
		return nil, err
	}
	var result UploadResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Serve) SaveVersion(request *pb.RIotSetVersion) error {
	// Get the project service client
	cc, ok := s.clients.Get(gutils.ISVIotOp)
	if !ok {
		return errors.New("fail to get iot_op service")
	}
	iotOp := pb.NewIotOpServiceClient(cc)
	if _, err := iotOp.SetVersion(context.TODO(), request); err != nil {
		return err
	}
	return nil
}

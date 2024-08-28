package serve

import (
	"context"
	"net/http"

	"github.com/Dcarbon/arch-proto/pb"
	"github.com/Dcarbon/go-shared/gutils"
	"github.com/Dcarbon/go-shared/libs/aidh"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

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

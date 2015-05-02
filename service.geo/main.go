package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"time"

	pb "github.com/harlow/go-micro-services/service.geo/proto"
	trace "github.com/harlow/go-micro-services/trace"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	port       = flag.Int("port", 10002, "The server port")
	jsonDBFile = flag.String("json_db_file", "data/locations.json", "A json file containing hotel locations")
	serverName = "service.geo"
)

type location struct {
	HotelID int32
	Point   *pb.Point
}

type geoServer struct {
	locations []location
}

// BoundedBox returns all hotels contained within a given rectangle.
func (s *geoServer) BoundedBox(ctx context.Context, rect *pb.Rectangle) (*pb.Reply, error) {
	md, _ := metadata.FromContext(ctx)
	t := trace.Tracer{TraceID: md["traceID"]}
	t.In(serverName, md["from"])
	defer t.Out(md["from"], serverName, time.Now())

	reply := new(pb.Reply)
	for _, loc := range s.locations {
		if inRange(loc.Point, rect) {
			reply.HotelIds = append(reply.HotelIds, loc.HotelID)
		}
	}

	return reply, nil
}

// loadLocations loads hotel locations from a JSON file.
func (s *geoServer) loadLocations(filePath string) {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to load file: %v", err)
	}
	if err := json.Unmarshal(file, &s.locations); err != nil {
		log.Fatalf("Failed to load hotels: %v", err)
	}
}

// inRange calculates if a point appears within a BoundingBox.
func inRange(point *pb.Point, rect *pb.Rectangle) bool {
	left := math.Min(float64(rect.Lo.Longitude), float64(rect.Hi.Longitude))
	right := math.Max(float64(rect.Lo.Longitude), float64(rect.Hi.Longitude))
	top := math.Max(float64(rect.Lo.Latitude), float64(rect.Hi.Latitude))
	bottom := math.Min(float64(rect.Lo.Latitude), float64(rect.Hi.Latitude))

	if float64(point.Longitude) >= left &&
		float64(point.Longitude) <= right &&
		float64(point.Latitude) >= bottom &&
		float64(point.Latitude) <= top {
		return true
	}
	return false
}

func newServer() *geoServer {
	s := new(geoServer)
	s.loadLocations(*jsonDBFile)
	return s
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterGeoServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}

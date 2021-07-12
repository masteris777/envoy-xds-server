//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/masteris777/go-watchfile"
	log "github.com/sirupsen/logrus"
	"github.com/stevesloka/envoy-xds-server/internal/processor"
	"github.com/stevesloka/envoy-xds-server/internal/server"
)

var (
	l log.FieldLogger

	watchDirectoryFileName string
	port                   uint
	basePort               uint
	mode                   string

	nodeID string
)

func init() {
	l = log.New()
	log.SetLevel(log.DebugLevel)

	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 9002, "xDS management server port")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "test-id", "Node ID")

	// Define the directory to watch for Envoy configuration files
	flag.StringVar(&watchDirectoryFileName, "watchDirectoryFileName", "./config/config.yaml", "full path to directory to watch for files")
}

func main() {
	flag.Parse()

	// Create a cache
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, l)

	// Create a processor
	proc := processor.NewProcessor(
		cache, nodeID, log.WithField("context", "processor"))

	// Create initial snapshot from file
	proc.ProcessFile(watchDirectoryFileName)

	go func() {
		// Run the xDS server
		ctx := context.Background()
		srv := serverv3.NewServer(ctx, cache, nil)
		server.RunServer(ctx, srv, port)
	}()

	// Start tracking file changes
	fileChangeNotification, fileCheckError := watchfile.Notify(watchDirectoryFileName);

	for {
		select {
			case err := <- fileCheckError:
				fmt.Println(err)
			case <- fileChangeNotification:
				log.Printf("Configuration file has been updated")
				proc.ProcessFile(watchDirectoryFileName)
		}
	}

}
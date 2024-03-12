package EKS

import (
	"awsx-api/log"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Appkube-awsx/awsx-common/authenticate"
	"github.com/Appkube-awsx/awsx-common/awsclient"
	"github.com/Appkube-awsx/awsx-common/model"
	"github.com/Appkube-awsx/awsx-getelementdetails/handler/EKS"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/spf13/cobra"
)

var (
	nodeUptimeAuthCache       sync.Map
	nodeUptimeClientCache     sync.Map
	nodeUptimeAuthCacheLock   sync.RWMutex
	nodeUptimeClientCacheLock sync.RWMutex
)

// NodeUptimeDataPoint represents each data point in the node uptime panel
type NodeUptimeDataPoint struct {
	Timestamp  time.Time `json:"Timestamp"`
	UptimeDays float64   `json:"UptimeDays"`
}

// NodeUptimePanelHandler handles the request for the node uptime panel data
func NodeUptimePanelHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse query parameters
	region := r.URL.Query().Get("zone")
	elementID := r.URL.Query().Get("elementId")
	elementAPIURL := r.URL.Query().Get("elementApiUrl")
	crossAccountRoleArn := r.URL.Query().Get("crossAccountRoleArn")
	externalID := r.URL.Query().Get("externalId")
	responseType := r.URL.Query().Get("responseType")
	instanceID := r.URL.Query().Get("instanceId")
	elementType := r.URL.Query().Get("elementType")
	startTime := r.URL.Query().Get("startTime")
	endTime := r.URL.Query().Get("endTime")

	commandParam := model.CommandParam{}

	if elementID != "" {
		commandParam.CloudElementId = elementID
		commandParam.CloudElementApiUrl = elementAPIURL
		commandParam.Region = region
	} else {
		commandParam.CrossAccountRoleArn = crossAccountRoleArn
		commandParam.ExternalId = externalID
		commandParam.Region = region
	}

	// Authenticate and get client authentication details
	clientAuth, err := nodeUptimeAuthenticateAndCache(commandParam)
	if err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %s", err), http.StatusInternalServerError)
		return
	}

	// Get CloudWatch client
	cloudwatchClient, err := nodeUptimeCloudwatchClientCache(*clientAuth)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cloudwatch client creation/store in cache failed: %s", err), http.StatusInternalServerError)
		return
	}

	if clientAuth != nil {
		// Prepare cobra command
		cmd := &cobra.Command{}
		cmd.PersistentFlags().StringVar(&elementID, "elementId", r.URL.Query().Get("elementId"), "Description of the cloudElementID flag")
		cmd.PersistentFlags().StringVar(&instanceID, "instanceId", r.URL.Query().Get("instanceId"), "Description of the instanceID flag")
		cmd.PersistentFlags().StringVar(&elementType, "elementType", r.URL.Query().Get("elementType"), "Description of the elementType flag")
		cmd.PersistentFlags().StringVar(&startTime, "startTime", r.URL.Query().Get("startTime"), "Description of the startTime flag")
		cmd.PersistentFlags().StringVar(&endTime, "endTime", r.URL.Query().Get("endTime"), "Description of the endTime flag")
		cmd.PersistentFlags().StringVar(&responseType, "responseType", r.URL.Query().Get("responseType"), "responseType flag - json/frame")

		// Get node uptime panel data
		jsonString, nodeUptimeData, err := EKS.GetNodeUptimePanel(cmd, clientAuth, cloudwatchClient)
		if err != nil {
			http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
			return
		}
		//fmt.Println(jsonString, nodeUptimeData)
		log.Infof("response type: %s", responseType)

		if responseType == "frame" {
			// Handle frame response
			err = json.NewEncoder(w).Encode(nodeUptimeData)
			if err != nil {
				http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
				return
			}
		} else {
			// Handle JSON response
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(jsonString))
			if err != nil {
				http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
				return
			}
		}
	}
}

// nodeUptimeAuthenticateAndCache authenticates and caches client details for node uptime panel
func nodeUptimeAuthenticateAndCache(commandParam model.CommandParam) (*model.Auth, error) {
	cacheKey := commandParam.CloudElementId

	nodeUptimeAuthCacheLock.Lock()
	defer nodeUptimeAuthCacheLock.Unlock()

	if auth, ok := nodeUptimeAuthCache.Load(cacheKey); ok {
		return auth.(*model.Auth), nil
	}

	_, clientAuth, err := authenticate.DoAuthenticate(commandParam)
	if err != nil {
		return nil, err
	}

	nodeUptimeAuthCache.Store(cacheKey, clientAuth)
	return clientAuth, nil
}

// nodeUptimeCloudwatchClientCache caches cloudwatch client for node uptime panel
func nodeUptimeCloudwatchClientCache(clientAuth model.Auth) (*cloudwatch.CloudWatch, error) {
	cacheKey := clientAuth.CrossAccountRoleArn

	nodeUptimeClientCacheLock.Lock()
	defer nodeUptimeClientCacheLock.Unlock()

	if client, ok := nodeUptimeClientCache.Load(cacheKey); ok {
		return client.(*cloudwatch.CloudWatch), nil
	}

	cloudWatchClient := awsclient.GetClient(clientAuth, awsclient.CLOUDWATCH).(*cloudwatch.CloudWatch)
	nodeUptimeClientCache.Store(cacheKey, cloudWatchClient)
	return cloudWatchClient, nil
}

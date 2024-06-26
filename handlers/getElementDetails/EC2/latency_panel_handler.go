package EC2

// import (
// 	"awsx-api/log"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"sync"

// 	"github.com/Appkube-awsx/awsx-common/authenticate"
// 	"github.com/Appkube-awsx/awsx-common/awsclient"
// 	"github.com/Appkube-awsx/awsx-common/model"
// 	"github.com/Appkube-awsx/awsx-getelementdetails/handler/EC2"
// 	"github.com/aws/aws-sdk-go/service/cloudwatch"
// 	"github.com/spf13/cobra"
// )

// type Ec2Latency struct {
// 	InboundTraffic  float64 `json:"inboundTraffic"`
// 	OutboundTraffic float64 `json:"outboundTraffic"`
// 	DataTransferred float64 `json:"dataTransferred"`
// 	Latency         float64 `json:"latency"`
// }

// type LatencyData struct {
// 	Latency float64 `json:"Latency"`
// }

// var (
// 	authCacheLatency       sync.Map
// 	clientCacheLatency     sync.Map
// 	authCacheLockLatency   sync.RWMutex
// 	clientCacheLockLatency sync.RWMutex
// )

// func GetLatencyPanel(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")

// 	region := r.URL.Query().Get("zone")
// 	elementId := r.URL.Query().Get("elementId")
// 	elementApiUrl := r.URL.Query().Get("cmdbApiUrl")
// 	elementType := r.URL.Query().Get("elementType")
// 	crossAccountRoleArn := r.URL.Query().Get("crossAccountRoleArn")
// 	externalId := r.URL.Query().Get("externalId")
// 	responseType := r.URL.Query().Get("responseType")
// 	instanceId := r.URL.Query().Get("instanceId")
// 	startTime := r.URL.Query().Get("startTime")
// 	endTime := r.URL.Query().Get("endTime")
// 	commandParam := model.CommandParam{}

// 	if elementId != "" {
// 		commandParam.CloudElementId = elementId
// 		commandParam.CloudElementApiUrl = elementApiUrl
// 		commandParam.Region = region
// 	} else {
// 		commandParam.CrossAccountRoleArn = crossAccountRoleArn
// 		commandParam.ExternalId = externalId
// 		commandParam.Region = region
// 	}
// 	clientAuth, err := authenticateAndCacheLatency(commandParam)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Authentication failed: %s", err), http.StatusInternalServerError)
// 		return
// 	}
// 	cloudwatchClient, err := cloudwatchClientCacheLatency(*clientAuth)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Cloudwatch client creation/store in cache failed: %s", err), http.StatusInternalServerError)
// 		return
// 	}
// 	if clientAuth != nil {
// 		cmd := &cobra.Command{}
// 		cmd.PersistentFlags().StringVar(&elementId, "elementId", r.URL.Query().Get("elementId"), "Description of the elementId flag")
// 		cmd.PersistentFlags().StringVar(&instanceId, "instanceId", r.URL.Query().Get("instanceId"), "Description of the instanceId flag")
// 		cmd.PersistentFlags().StringVar(&elementType, "elementType", r.URL.Query().Get("elementType"), "Description of the elementType flag")
// 		cmd.PersistentFlags().StringVar(&startTime, "startTime", r.URL.Query().Get("startTime"), "Description of the startTime flag")
// 		cmd.PersistentFlags().StringVar(&endTime, "endTime", r.URL.Query().Get("endTime"), "Description of the endTime flag")
// 		cmd.PersistentFlags().StringVar(&responseType, "responseType", r.URL.Query().Get("responseType"), "responseType flag - json/frame")
// 		jsonString, cloudwatchMetricData, err := EC2.GetLatencyPanel(cmd, clientAuth, cloudwatchClient)
// 		if err != nil {
// 			http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
// 			return
// 		}
// 		log.Infof("response type :" + responseType)
// 		if responseType == "frame" {
// 			err = json.NewEncoder(w).Encode(cloudwatchMetricData)
// 			if err != nil {
// 				http.Error(w, fmt.Sprintf("Exception: %s ", err), http.StatusInternalServerError)
// 				return
// 			}
// 		} else {
// 			var data LatencyData
// 			err := json.Unmarshal([]byte(jsonString), &data)
// 			if err != nil {
// 				http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
// 				return
// 			}

// 			jsonBytes, err := json.Marshal(data)
// 			if err != nil {
// 				http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
// 				return
// 			}
// 			w.Header().Set("Content-Type", "application/json")
// 			_, err = w.Write(jsonBytes)
// 			if err != nil {
// 				http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
// 				return
// 			}
// 		}
// 	}
// }

// func authenticateAndCacheLatency(commandParam model.CommandParam) (*model.Auth, error) {
// 	cacheKey := commandParam.CloudElementId

// 	authCacheLockLatency.Lock()
// 	if auth, ok := authCacheLatency.Load(cacheKey); ok {
// 		log.Infof("client credentials found in cache")
// 		authCacheLockLatency.Unlock()
// 		return auth.(*model.Auth), nil
// 	}

// 	// If not in cache, perform authentication
// 	log.Infof("getting client credentials from vault/db")
// 	_, clientAuth, err := authenticate.DoAuthenticate(commandParam)
// 	if err != nil {
// 		return nil, err
// 	}

// 	authCacheLatency.Store(cacheKey, clientAuth)
// 	authCacheLockLatency.Unlock()

// 	return clientAuth, nil
// }

// func cloudwatchClientCacheLatency(clientAuth model.Auth) (*cloudwatch.CloudWatch, error) {
// 	cacheKey := clientAuth.CrossAccountRoleArn

// 	clientCacheLockLatency.Lock()
// 	if client, ok := clientCacheLatency.Load(cacheKey); ok {
// 		log.Infof("cloudwatch client found in cache for given cross acount role: %s", cacheKey)
// 		clientCacheLockLatency.Unlock()
// 		return client.(*cloudwatch.CloudWatch), nil
// 	}

// 	// If not in cache, create new cloud watch client
// 	log.Infof("creating new cloudwatch client for given cross acount role: %s", cacheKey)
// 	cloudWatchClient := awsclient.GetClient(clientAuth, awsclient.CLOUDWATCH).(*cloudwatch.CloudWatch)

// 	clientCacheLatency.Store(cacheKey, cloudWatchClient)
// 	clientCacheLockLatency.Unlock()

// 	return cloudWatchClient, nil
// }

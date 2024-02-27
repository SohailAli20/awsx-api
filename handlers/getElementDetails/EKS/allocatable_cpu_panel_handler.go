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

// Define package level variables for allocatable CPU panel
var (
	allocatableCPUAuthCache       sync.Map
	allocatableCPUClientCache     sync.Map
	allocatableCPUAuthCacheLock   sync.RWMutex
	allocatableCPUClientCacheLock sync.RWMutex
)

// GetEKSAllocatableCPUPanel handles the request for the allocatable CPU panel data
func GetEKSAllocatableCPUPanel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse query parameters
	region := r.URL.Query().Get("zone")
	elementId := r.URL.Query().Get("elementId")
	elementApiUrl := r.URL.Query().Get("elementApiUrl")
	crossAccountRoleArn := r.URL.Query().Get("crossAccountRoleArn")
	externalId := r.URL.Query().Get("externalId")
	responseType := r.URL.Query().Get("responseType")
	instanceId := r.URL.Query().Get("instanceId")
	elementType := r.URL.Query().Get("elementType")
	startTime := r.URL.Query().Get("startTime")
	endTime := r.URL.Query().Get("endTime")

	commandParam := model.CommandParam{}

	if elementId != "" {
		commandParam.CloudElementId = elementId
		commandParam.CloudElementApiUrl = elementApiUrl
		commandParam.Region = region
	} else {
		commandParam.CrossAccountRoleArn = crossAccountRoleArn
		commandParam.ExternalId = externalId
		commandParam.Region = region
	}

	type allocatableResult struct {
		RawData []struct {
			Timestamp time.Time `json:"Timestamp"`
			Value     float64   `json:"AllocatableCPU"`
		} `json:"RawData"`
	}

	// Authenticate and get client authentication details
	clientAuth, err := allocatedAuthenticateAndCache(commandParam)
	if err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %s", err), http.StatusInternalServerError)
		return
	}

	// Get CloudWatch client
	cloudwatchClient, err := allocatedCloudwatchClientCache(*clientAuth)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cloudwatch client creation/store in cache failed: %s", err), http.StatusInternalServerError)
		return
	}

	if clientAuth != nil {
		// Prepare cobra command
		cmd := &cobra.Command{}
		cmd.PersistentFlags().StringVar(&elementId, "elementId", r.URL.Query().Get("elementId"), "Description of the cloudElementID flag")
		cmd.PersistentFlags().StringVar(&instanceId, "instanceId", r.URL.Query().Get("instanceId"), "Description of the instanceID flag")
		cmd.PersistentFlags().StringVar(&elementType, "elementType", r.URL.Query().Get("elementType"), "Description of the elementType flag")
		cmd.PersistentFlags().StringVar(&startTime, "startTime", r.URL.Query().Get("startTime"), "Description of the startTime flag")
		cmd.PersistentFlags().StringVar(&endTime, "endTime", r.URL.Query().Get("endTime"), "Description of the endTime flag")
		cmd.PersistentFlags().StringVar(&responseType, "responseType", r.URL.Query().Get("responseType"), "responseType flag - json/frame")

		// Get allocatable CPU panel data
		jsonString, cloudwatchMetricData, err := EKS.GetAllocatableCPUData(cmd, clientAuth, cloudwatchClient)
		fmt.Println(jsonString)
		if err != nil {
			http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
			return
		}
		log.Infof("response type :" + responseType)
		if responseType == "frame" {
			err = json.NewEncoder(w).Encode(cloudwatchMetricData)
			if err != nil {
				http.Error(w, fmt.Sprintf("Exception: %s ", err), http.StatusInternalServerError)
				return
			}
		} else {
			var data allocatableResult
			err := json.Unmarshal([]byte(jsonString), &data)
			if err != nil {
				http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
				return
			}

			jsonBytes, err := json.Marshal(data)
			fmt.Println(data)
			if err != nil {
				http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(jsonBytes)
			if err != nil {
				http.Error(w, fmt.Sprintf("Exception: %s", err), http.StatusInternalServerError)
				return
			}
		}
	}
}

// authenticateAndCache authenticates and caches client details
func allocatedAuthenticateAndCache(commandParam model.CommandParam) (*model.Auth, error) {
	cacheKey := commandParam.CloudElementId

	allocatableCPUAuthCacheLock.Lock()
	defer allocatableCPUAuthCacheLock.Unlock()

	if auth, ok := allocatableCPUAuthCache.Load(cacheKey); ok {
		return auth.(*model.Auth), nil
	}

	_, clientAuth, err := authenticate.DoAuthenticate(commandParam)
	if err != nil {
		return nil, err
	}

	allocatableCPUAuthCache.Store(cacheKey, clientAuth)
	return clientAuth, nil
}

// cloudwatchClientCache caches cloudwatch client
func allocatedCloudwatchClientCache(clientAuth model.Auth) (*cloudwatch.CloudWatch, error) {
	cacheKey := clientAuth.CrossAccountRoleArn

	allocatableCPUClientCacheLock.Lock()
	defer allocatableCPUClientCacheLock.Unlock()

	if client, ok := allocatableCPUClientCache.Load(cacheKey); ok {
		return client.(*cloudwatch.CloudWatch), nil
	}

	cloudWatchClient := awsclient.GetClient(clientAuth, awsclient.CLOUDWATCH).(*cloudwatch.CloudWatch)
	allocatableCPUClientCache.Store(cacheKey, cloudWatchClient)
	return cloudWatchClient, nil
}

// package EKS

// import (
// 	"awsx-api/log"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	// "sync"
// 	"time"

// 	// "github.com/Appkube-awsx/awsx-common/authenticate"
// 	// "github.com/Appkube-awsx/awsx-common/awsclient"
// 	"github.com/Appkube-awsx/awsx-common/model"
// 	"github.com/Appkube-awsx/awsx-getelementdetails/handler/EKS"
// 	// "github.com/aws/aws-sdk-go/service/cloudwatch"
// 	"github.com/spf13/cobra"
// )

// type allocatableResult struct {
// 	RawData []struct {
// 		Timestamp time.Time
// 		Value     float64
// 	} `json:"RawData"`
// }

// // var (
// // 	authCache       sync.Map
// // 	clientCache     sync.Map
// // 	authCacheLock   sync.RWMutex
// // 	clientCacheLock sync.RWMutex
// // 	//authCacheLock sync.Mutex
// // )

// func GetAllocatableCpuPanel(w http.ResponseWriter, r *http.Request) {
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
// 	clientAuth, err := authenticateAndCache(commandParam)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Authentication failed: %s", err), http.StatusInternalServerError)
// 		return
// 	}
// 	cloudwatchClient, err := cloudwatchClientCache(*clientAuth)
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
// 		jsonString, cloudwatchMetricData, err := EKS.GetAllocatableCPUData(cmd, clientAuth, cloudwatchClient)
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
// 			var data allocatableResult
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

// // func authenticateAndCache(commandParam model.CommandParam) (*model.Auth, error) {
// // 	cacheKey := commandParam.CloudElementId

// // 	authCacheLock.Lock()
// // 	if auth, ok := authCache.Load(cacheKey); ok {
// // 		log.Infof("client credentials found in cache")
// // 		authCacheLock.Unlock()
// // 		return auth.(*model.Auth), nil
// // 	}

// // 	// If not in cache, perform authentication
// // 	log.Infof("getting client credentials from vault/db")
// // 	_, clientAuth, err := authenticate.DoAuthenticate(commandParam)
// // 	if err != nil {
// // 		return nil, err
// // 	}

// // 	authCache.Store(cacheKey, clientAuth)
// // 	authCacheLock.Unlock()

// // 	return clientAuth, nil
// // }

// // func cloudwatchClientCache(clientAuth model.Auth) (*cloudwatch.CloudWatch, error) {
// // 	cacheKey := clientAuth.CrossAccountRoleArn

// // 	clientCacheLock.Lock()
// // 	if client, ok := clientCache.Load(cacheKey); ok {
// // 		log.Infof("cloudwatch client found in cache for given cross acount role: %s", cacheKey)
// // 		clientCacheLock.Unlock()
// // 		return client.(*cloudwatch.CloudWatch), nil
// // 	}

// // 	// If not in cache, create new cloud watch client
// // 	log.Infof("creating new cloudwatch client for given cross acount role: %s", cacheKey)
// // 	cloudWatchClient := awsclient.GetClient(clientAuth, awsclient.CLOUDWATCH).(*cloudwatch.CloudWatch)

// // 	clientCache.Store(cacheKey, cloudWatchClient)
// // 	clientCacheLock.Unlock()

// // 	return cloudWatchClient, nil
// // }

package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Hari-Kiri/goalApplicationSettingsLoader"
	"github.com/Hari-Kiri/goalHash"
	"github.com/Hari-Kiri/goalJson"
	"github.com/Hari-Kiri/goalMakeHandler"
	"github.com/Hari-Kiri/goalMySql"
)

func main() {
	// Load settings
	loadApplicationSettings, errorLoadApplicationSettings := goalApplicationSettingsLoader.LoadSettings()
	if errorLoadApplicationSettings != nil {
		log.Panic("[error] kbackend failed to start with the following reason:", errorLoadApplicationSettings)
	}
	// Create new database handler
	dbHandler, errorDBHandler := goalMySql.Initialize(true)
	if errorDBHandler != nil {
		log.Panic("[error] kbackend failed to create new database handler with the following reason:",
			errorDBHandler)
	}
	// Test database connection
	testDBConnection, errorTestDBConnection := goalMySql.PingDatabase(dbHandler)
	if errorTestDBConnection != nil {
		log.Panic("[error] kbackend failed to connect to database  with the following reason:",
			errorDBHandler)
	}
	log.Output(1, "[info] MySql connection: "+fmt.Sprintf("%v", testDBConnection))
	log.Output(1, "[info] Starting webserver")
	// Handle web root request
	goalMakeHandler.HandleRequest(rootHandler, "/")
	// Handle test page request (its just for testing webserver online or not)
	goalMakeHandler.HandleRequest(testHandler, "/test")
	// Handle login request
	goalMakeHandler.HandleRequest(loginHandler, "/login")
	// Handle merchs list request
	goalMakeHandler.HandleRequest(merchsHandler, "/merchs")
	// Handle merchs update request
	goalMakeHandler.HandleRequest(updateMerchsQuantityHandler, "/merchsupdate")
	// Handle all merchs list request
	goalMakeHandler.HandleRequest(allMerchsHandler, "/allmerchs")
	// Handle purchase merchs request
	goalMakeHandler.HandleRequest(purchaseHandler, "/purchase")
	// Run HTTP server
	goalMakeHandler.Serve(loadApplicationSettings.Settings.Name, loadApplicationSettings.Settings.Port)
}

// Web root handler
func rootHandler(responseWriter http.ResponseWriter, request *http.Request) {
	// Redirect to home page
	http.Redirect(responseWriter, request, "/test", http.StatusFound)
	log.Output(1, "[info] Webroot redirect to url path ["+request.URL.Path+"], requested from "+request.RemoteAddr)
}

// Test page handler
func testHandler(responseWriter http.ResponseWriter, request *http.Request) {
	// Http ok response
	okResponse, _ := goalJson.JsonEncode(map[string]interface{}{
		"response": true,
		"code":     200,
		"message":  "Go net/http webserver online"},
		false)
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.Write([]byte(okResponse))
	log.Output(1, "[info] Serving test page ["+request.URL.Path+"], requested from "+request.RemoteAddr)
}

// Handle http request body
func handleRequestBody(request *http.Request) (map[string]interface{}, error) {
	// Read http request body
	requestBody, errorRequestBody := ioutil.ReadAll(request.Body)
	if len(requestBody) == 0 {
		return nil, fmt.Errorf("request body empty: %s", errorRequestBody)
	}
	if errorRequestBody != nil {
		return nil, errorRequestBody
	}
	// Decode encrypted data from base64 to byte array
	decodeData, errorDecodeData := base64.StdEncoding.DecodeString(string(requestBody))
	if errorDecodeData != nil {
		return nil, fmt.Errorf("base64 Data decoding failed: %q", errorDecodeData)
	}
	// Decode json request body
	// JsonDecode will return error if fail
	// So we can return out single value
	return goalJson.JsonDecode(string(decodeData))
}

// Login handler
func loginHandler(responseWriter http.ResponseWriter, request *http.Request) {
	/* Handle request body */
	requestBody, errorRequestBody := handleRequestBody(request)
	if errorRequestBody != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     406,
			"message":  "request body empty"},
			false)
		http.Error(responseWriter, errorResponse, http.StatusNotAcceptable)
		log.Output(1, "[error] loginHandler() Error read http body for response ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+errorRequestBody.Error())
		return
	}
	/* Check account credential from database ecomm.users */
	userCredential, errorGetUserCredential := checkUserAccount(
		requestBody["account"].(map[string]interface{})["user"].(string),
		goalHash.Sha256(requestBody["account"].(map[string]interface{})["password"].(string)))
	if errorGetUserCredential != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     404,
			"message":  "account not authenticated"},
			false)
		http.Error(responseWriter, errorResponse, http.StatusNotFound)
		log.Output(1, "[error] loginHandler() cannot find account with username: "+
			requestBody["account"].(map[string]interface{})["user"].(string)+
			", response for ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+errorGetUserCredential.Error())
		return
	}
	/* Create response to client */
	okResponse, _ := goalJson.JsonEncode(map[string]interface{}{
		"response": true,
		"code":     200,
		"message": []map[string]interface{}{
			{
				"status": "login success",
				"userId": userCredential["id"],
				"level":  userCredential["level"],
			},
		},
	}, false)
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Header().Set("Content-Type", "text/plain")
	responseWriter.Write([]byte(base64.StdEncoding.EncodeToString([]byte(okResponse))))
	log.Output(1, "[info] Serving login request ["+request.URL.Path+"], requested from "+request.RemoteAddr+
		", account authenticated, user id: "+fmt.Sprintf("%s", userCredential["id"]))
}

// Check user account
func checkUserAccount(username string, password string) (map[string]interface{}, error) {
	// Create new database handler
	dbHandler, errorDBHandler := goalMySql.Initialize(true)
	if errorDBHandler != nil {
		return nil, errorDBHandler
	}
	// Test connection to database
	pingDatabase, errorPingDatabase := goalMySql.PingDatabase(dbHandler)
	if !pingDatabase && errorPingDatabase != nil {
		return nil, errorPingDatabase
	}
	if !pingDatabase && errorPingDatabase == nil {
		return nil, fmt.Errorf("cannot connect to MySql node")
	}
	log.Output(1, "[info] MySql connected")
	// Check login credential
	log.Output(1, "[info] Check account, username: "+username)
	querySelectMyuser, errorQuerySelectMyuser := goalMySql.Select(
		dbHandler,
		"id, level",
		"ecomm.users",
		"WHERE name = ? AND password = ?",
		username,
		password)
	if errorQuerySelectMyuser != nil {
		return nil, errorQuerySelectMyuser
	}
	// User not authenticated
	if len(querySelectMyuser) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	// User authenticated
	return querySelectMyuser[0], nil
}

// Merchs list handler
func merchsHandler(responseWriter http.ResponseWriter, request *http.Request) {
	/* Handle request body */
	requestBody, errorRequestBody := handleRequestBody(request)
	if errorRequestBody != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     406,
			"message":  "request body empty"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotAcceptable)
		log.Output(1, "[error] merchsHandler() Error read http body for response ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+errorRequestBody.Error())
		return
	}
	/* Check account credential from database ecomm.users */
	userCredential, errorGetUserCredential := checkUserAccount(
		requestBody["account"].(map[string]interface{})["user"].(string),
		goalHash.Sha256(requestBody["account"].(map[string]interface{})["password"].(string)))
	if errorGetUserCredential != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     404,
			"message":  "account not authenticated"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotFound)
		log.Output(1, "[error] merchsHandler() cannot find account with username: "+
			requestBody["account"].(map[string]interface{})["user"].(string)+
			", response for ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+errorGetUserCredential.Error())
		return
	}
	if userCredential["level"] != "SELLER" {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     406,
			"message":  "account not seller"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotAcceptable)
		log.Output(1, "[error] merchsHandler() cannot get merchs list with username: "+
			requestBody["account"].(map[string]interface{})["user"].(string)+
			", response for ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+"account level is "+
			userCredential["level"].(string))
		return
	}
	/* Get merchs from database */
	merchsList, errorGetMerchsList := getMerchs(userCredential["id"].(string))
	if errorGetMerchsList != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     404,
			"message":  "cannot get merchs list"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotFound)
		log.Output(1, "[error] merchsHandler() cannot get merchs list with username: "+
			requestBody["account"].(map[string]interface{})["user"].(string)+
			", response for ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+errorGetMerchsList.Error())
		return
	}
	/* Create response to client */
	okResponse, _ := goalJson.JsonEncode(map[string]interface{}{
		"response": true,
		"code":     200,
		"message": []map[string]interface{}{
			{
				"status": "listing merchs success",
				"merchs": merchsList,
			},
		},
	}, false)
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Header().Set("Content-Type", "text/plain")
	responseWriter.Write([]byte(base64.StdEncoding.EncodeToString([]byte(okResponse))))
	log.Output(1, "[info] Serving merchs request ["+request.URL.Path+"], requested from "+request.RemoteAddr+
		", account authenticated, user id: "+fmt.Sprintf("%s", userCredential["id"]))
}

// Get merchs
func getMerchs(userId string) ([]map[string]interface{}, error) {
	// Create new database handler
	dbHandler, errorDBHandler := goalMySql.Initialize(true)
	if errorDBHandler != nil {
		return nil, errorDBHandler
	}
	// Test connection to database
	pingDatabase, errorPingDatabase := goalMySql.PingDatabase(dbHandler)
	if !pingDatabase && errorPingDatabase != nil {
		return nil, errorPingDatabase
	}
	if !pingDatabase && errorPingDatabase == nil {
		return nil, fmt.Errorf("cannot connect to MySql node")
	}
	log.Output(1, "[info] MySql connected")
	// Get merchs from database
	log.Output(1, "[info] Get merchs list, seller id: "+userId)
	querySelectMerchs, errorQuerySelectMerchs := goalMySql.Select(
		dbHandler,
		"id, name, quantity",
		"ecomm.goods",
		"WHERE seller_id = ?",
		userId,
	)
	if errorQuerySelectMerchs != nil {
		return nil, errorQuerySelectMerchs
	}
	if len(querySelectMerchs) == 0 {
		return nil, fmt.Errorf("merchs empty")
	}
	return querySelectMerchs, nil
}

// Update merchs handler
func updateMerchsQuantityHandler(responseWriter http.ResponseWriter, request *http.Request) {
	/* Handle request body */
	requestBody, errorRequestBody := handleRequestBody(request)
	if errorRequestBody != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     406,
			"message":  "request body empty"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotAcceptable)
		log.Output(1, "[error] updateMerchsQuantityHandler() Error read http body for response ["+
			request.URL.Path+"], requested from "+request.RemoteAddr+": "+errorRequestBody.Error())
		return
	}
	/* Check account credential from database ecomm.users */
	userCredential, errorGetUserCredential := checkUserAccount(
		requestBody["account"].(map[string]interface{})["user"].(string),
		goalHash.Sha256(requestBody["account"].(map[string]interface{})["password"].(string)))
	if errorGetUserCredential != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     404,
			"message":  "account not authenticated"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotFound)
		log.Output(1, "[error] updateMerchsQuantityHandler() cannot find account with username: "+
			requestBody["account"].(map[string]interface{})["user"].(string)+
			", response for ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+errorGetUserCredential.Error())
		return
	}
	/* Update merchs */
	// Convert user id from mysql select to integer
	userId, _ := strconv.Atoi(userCredential["id"].(string))
	updateMerchs, errorUpdateMerchs := updateMerchsQuantity(
		userId,
		int(requestBody["update"].(map[string]interface{})["merchsId"].(float64)),
		int(requestBody["update"].(map[string]interface{})["quantity"].(float64)),
	)
	if errorUpdateMerchs != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     404,
			"message":  "merchs update failed"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotFound)
		log.Output(1, "[error] updateMerchsQuantityHandler() cannot update merchs: "+errorUpdateMerchs.Error())
		return
	}
	/* Create response to client */
	okResponse, _ := goalJson.JsonEncode(map[string]interface{}{
		"response": true,
		"code":     200,
		"message": []map[string]interface{}{
			{
				"status": "update merchs success",
				"update": fmt.Sprintf("%d", updateMerchs) + " rows updated",
			},
		},
	}, false)
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Header().Set("Content-Type", "text/plain")
	responseWriter.Write([]byte(base64.StdEncoding.EncodeToString([]byte(okResponse))))
	log.Output(1, "[info] Serving update merchs request ["+request.URL.Path+"], requested from "+request.RemoteAddr+
		", account authenticated, user id: "+fmt.Sprintf("%s", userCredential["id"]))
}

// Update merchs quantity
func updateMerchsQuantity(userId int, merchsId int, quantity int) (int, error) {
	// Create new database handler
	dbHandler, errorDBHandler := goalMySql.Initialize(true)
	if errorDBHandler != nil {
		return 0, errorDBHandler
	}
	// Test connection to database
	pingDatabase, errorPingDatabase := goalMySql.PingDatabase(dbHandler)
	if !pingDatabase && errorPingDatabase != nil {
		return 0, errorPingDatabase
	}
	if !pingDatabase && errorPingDatabase == nil {
		return 0, fmt.Errorf("cannot connect to MySql node")
	}
	log.Output(1, "[info] MySql connected")
	// Update merchs quantity
	log.Output(1, "[info] seller id "+fmt.Sprintf("%d", userId)+" update merchs id "+fmt.Sprintf("%d", merchsId)+
		" quantity to "+fmt.Sprintf("%d", quantity))
	updateQuantity, errorUpdatingQuantity := goalMySql.Update(
		dbHandler,
		"ecomm.goods",
		"quantity = ?, lup = ?",
		"WHERE id = ? AND seller_id = ?",
		quantity,
		time.Now(),
		merchsId,
		userId,
	)
	if errorUpdatingQuantity != nil {
		return 0, errorUpdatingQuantity
	}
	if updateQuantity == 0 {
		return 0, fmt.Errorf("update quantity failed")
	}
	return updateQuantity, nil
}

// List all merchs
func allMerchsHandler(responseWriter http.ResponseWriter, request *http.Request) {
	/* Handle request body */
	requestBody, errorRequestBody := handleRequestBody(request)
	if errorRequestBody != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     406,
			"message":  "request body empty"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotAcceptable)
		log.Output(1, "[error] allMerchsHandler() Error read http body for response ["+
			request.URL.Path+"], requested from "+request.RemoteAddr+": "+errorRequestBody.Error())
		return
	}
	/* Check account credential from database ecomm.users */
	userCredential, errorGetUserCredential := checkUserAccount(
		requestBody["account"].(map[string]interface{})["user"].(string),
		goalHash.Sha256(requestBody["account"].(map[string]interface{})["password"].(string)))
	if errorGetUserCredential != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     404,
			"message":  "account not authenticated"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotFound)
		log.Output(1, "[error] allMerchsHandler() cannot find account with username: "+
			requestBody["account"].(map[string]interface{})["user"].(string)+
			", response for ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+errorGetUserCredential.Error())
		return
	}
	if userCredential["level"] != "BUYER" {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     406,
			"message":  "account not buyer"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotAcceptable)
		log.Output(1, "[error] allMerchsHandler() cannot get merchs list with username: "+
			requestBody["account"].(map[string]interface{})["user"].(string)+
			", response for ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+"account level is "+
			userCredential["level"].(string))
		return
	}
	/* Get merchs from database */
	allMerchsList, errorGetAllMerchsList := getAllMerchs(userCredential["id"].(string))
	if errorGetAllMerchsList != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     404,
			"message":  "cannot get merchs list"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotFound)
		log.Output(1, "[error] allMerchsHandler() cannot get merchs list with username: "+
			requestBody["account"].(map[string]interface{})["user"].(string)+
			", response for ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+errorGetAllMerchsList.Error())
		return
	}
	/* Create response to client */
	okResponse, _ := goalJson.JsonEncode(map[string]interface{}{
		"response": true,
		"code":     200,
		"message": []map[string]interface{}{
			{
				"status": "listing merchs success",
				"merchs": allMerchsList,
			},
		},
	}, false)
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Header().Set("Content-Type", "text/plain")
	responseWriter.Write([]byte(base64.StdEncoding.EncodeToString([]byte(okResponse))))
	log.Output(1, "[info] Serving all merchs request ["+request.URL.Path+"], requested from "+request.RemoteAddr+
		", account authenticated, user id: "+fmt.Sprintf("%s", userCredential["id"]))
}

// Get all merchs data
func getAllMerchs(userId string) ([]map[string]interface{}, error) {
	// Create new database handler
	dbHandler, errorDBHandler := goalMySql.Initialize(true)
	if errorDBHandler != nil {
		return nil, errorDBHandler
	}
	// Test connection to database
	pingDatabase, errorPingDatabase := goalMySql.PingDatabase(dbHandler)
	if !pingDatabase && errorPingDatabase != nil {
		return nil, errorPingDatabase
	}
	if !pingDatabase && errorPingDatabase == nil {
		return nil, fmt.Errorf("cannot connect to MySql node")
	}
	log.Output(1, "[info] MySql connected")
	// Get merchs from database
	log.Output(1, "[info] Get all merchs list, seller id: "+userId)
	querySelectMerchs, errorQuerySelectMerchs := goalMySql.Select(
		dbHandler,
		"id, name, seller_id, quantity",
		"ecomm.goods",
		"WHERE quantity <> 0",
	)
	if errorQuerySelectMerchs != nil {
		return nil, errorQuerySelectMerchs
	}
	if len(querySelectMerchs) == 0 {
		return nil, fmt.Errorf("merchs empty")
	}
	return querySelectMerchs, nil
}

// Purchase handler
func purchaseHandler(responseWriter http.ResponseWriter, request *http.Request) {
	/* Handle request body */
	requestBody, errorRequestBody := handleRequestBody(request)
	if errorRequestBody != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     406,
			"message":  "request body empty"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotAcceptable)
		log.Output(1, "[error] purchase() Error read http body for response ["+
			request.URL.Path+"], requested from "+request.RemoteAddr+": "+errorRequestBody.Error())
		return
	}
	/* Check account credential from database ecomm.users */
	userCredential, errorGetUserCredential := checkUserAccount(
		requestBody["account"].(map[string]interface{})["user"].(string),
		goalHash.Sha256(requestBody["account"].(map[string]interface{})["password"].(string)))
	if errorGetUserCredential != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     404,
			"message":  "account not authenticated"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotFound)
		log.Output(1, "[error] purchase() cannot find account with username: "+
			requestBody["account"].(map[string]interface{})["user"].(string)+
			", response for ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+errorGetUserCredential.Error())
		return
	}
	if userCredential["level"] != "BUYER" {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     406,
			"message":  "account not buyer"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotAcceptable)
		log.Output(1, "[error] purchase() cannot get merchs list with username: "+
			requestBody["account"].(map[string]interface{})["user"].(string)+
			", response for ["+request.URL.Path+
			"], requested from "+request.RemoteAddr+": "+"account level is "+
			userCredential["level"].(string))
		return
	}
	/* Insert data to purchase table */
	userId, _ := strconv.Atoi(userCredential["id"].(string))
	purchase, errorPurchase := purchase(
		userId,
		int(requestBody["purchase"].(map[string]interface{})["merchsId"].(float64)),
		requestBody["purchase"].(map[string]interface{})["purchaseItem"].(string),
		int(requestBody["purchase"].(map[string]interface{})["sellerId"].(float64)),
		int(requestBody["purchase"].(map[string]interface{})["quantity"].(float64)),
	)
	if errorPurchase != nil {
		// Http error response
		errorResponse, _ := goalJson.JsonEncode(map[string]interface{}{
			"response": false,
			"code":     404,
			"message":  "merchs purchase failed"},
			false)
		http.Error(responseWriter, base64.StdEncoding.EncodeToString([]byte(errorResponse)), http.StatusNotFound)
		log.Output(1, "[error] purchase() cannot purchase merchs: "+errorPurchase.Error())
		return
	}
	/* Create response to client */
	okResponse, _ := goalJson.JsonEncode(map[string]interface{}{
		"response": true,
		"code":     200,
		"message": []map[string]interface{}{
			{
				"status": "purchase merchs success",
				"merchs": purchase,
			},
		},
	}, false)
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Header().Set("Content-Type", "text/plain")
	responseWriter.Write([]byte(base64.StdEncoding.EncodeToString([]byte(okResponse))))
	log.Output(1, "[info] Serving purchase merchs request ["+request.URL.Path+"], requested from "+request.RemoteAddr+
		", account authenticated, user id: "+fmt.Sprintf("%s", userCredential["id"]))
}

// Inser data to purchase table
func purchase(buyerId int, merchsId int, purchaseItem string, sellerId int, quantity int) (int, error) {
	// Create new database handler
	dbHandler, errorDBHandler := goalMySql.Initialize(true)
	if errorDBHandler != nil {
		return 0, errorDBHandler
	}
	// Test connection to database
	pingDatabase, errorPingDatabase := goalMySql.PingDatabase(dbHandler)
	if !pingDatabase && errorPingDatabase != nil {
		return 0, errorPingDatabase
	}
	if !pingDatabase && errorPingDatabase == nil {
		return 0, fmt.Errorf("cannot connect to MySql node")
	}
	log.Output(1, "[info] MySql connected")
	// Insert data
	insert, errorInsert := goalMySql.Insert(
		dbHandler,
		"ecomm.purchases",
		"buyer_id, merchs_id, purchase_item, seller_id, quantity, lup",
		buyerId,
		merchsId,
		purchaseItem,
		sellerId,
		quantity,
		time.Now(),
	)
	if errorInsert != nil {
		return 0, errorInsert
	}
	if insert == 0 {
		return insert, fmt.Errorf("new purchase failed")
	}
	return insert, nil
}


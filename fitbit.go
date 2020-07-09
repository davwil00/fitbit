package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go"
	"github.com/joho/godotenv"
)

const (
	callbackUrl string = "http://localhost:3000/fitbit"
)

func initClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	}

	return &http.Client{Transport: tr}
}

func main() {
	fmt.Println("Running")
	godotenv.Load("vars.env")
	client := initClient()
	accessToken := fetchToken(client)
	date := time.Now().AddDate(0, 0, -1)
	data := fetchHeartRateData(client, accessToken, date)
	writeToInflux(date, data.ActivitiesHeartIntraday.Dataset)
}

func writeToInflux(date time.Time, data []FitbitHeartIntradayData) {
	fmt.Println("Writing data size: ", len(data))
	
	token := os.Getenv("INFLUXDB_TOKEN")
	url := os.Getenv("INFLUXDB_URL")
	client := influxdb2.NewClient(url, token)

	org := os.Getenv("INFLUXDB_USER")
	bucket := os.Getenv("INFLUXDB_BUCKET")

	writeApi := client.WriteApi(org, bucket)

	for _, d := range data {
		dataTime := strings.Split(d.Time, ":")
		hour, _ := strconv.Atoi(dataTime[0])
		minute, _ := strconv.Atoi(dataTime[1])
		second, _ := strconv.Atoi(dataTime[2])
		pointTime := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, second, 0, time.UTC)
		p := influxdb2.NewPointWithMeasurement("activity").
			AddTag("unit", "bpm").
			AddField("count", d.Value).
			SetTime(pointTime)

		writeApi.WritePoint(p)
	}
	writeApi.Flush()
	client.Close()
}

func fetchHeartRateData(client *http.Client, accessToken string, date time.Time) FitbitHeartRateResponse {
	fmt.Println("Fetching heart rate data")
	url := fmt.Sprintf("https://api.fitbit.com/1/user/-/activities/heart/date/%s/1d/1sec/time/00:00/23:59.json", date.Format("2006-01-02"))
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("Error retrieving heart rate data: %s\n", err)
		os.Exit(0)
	}

	defer resp.Body.Close()

	var jsonBody FitbitHeartRateResponse
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &jsonBody)

	return jsonBody
}

func fetchToken(client *http.Client) string {
	fmt.Println("Fetching token")
	clientId := os.Getenv("FITBIT_CLIENT_ID")
	clientSecret := os.Getenv("FITBIT_CLIENT_SECRET")
	callbackUrl := os.Getenv("FITBIT_CALLBACK_URL")
	refreshCode := os.Getenv("FITBIT_REFRESH_CODE")
	refreshToken, err := ioutil.ReadFile("refreshToken.txt")
	var url string

	if len(refreshToken) > 0 {
		url = fmt.Sprintf("https://api.fitbit.com/oauth2/token?client_id=%s&grant_type=refresh_token&refresh_token=%s", clientId, refreshToken)
	} else {
		url = fmt.Sprintf("https://api.fitbit.com/oauth2/token?client_id=%s&grant_type=authorization_code&redirect_uri=%s&code=%s", clientId, callbackUrl, refreshCode)
	}

	req, _ := http.NewRequest("POST", url, nil)

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientId, clientSecret)

	resp, err := client.Do(req)

	if err != nil || resp.StatusCode != 200 {
		fmt.Printf("Error retrieving token: %s\n", err)
		fmt.Printf(resp.Status)
		respBody, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("%s", respBody)
		os.Exit(0)
	}

	defer resp.Body.Close()

	var jsonBody FitbitTokenResponse
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &jsonBody)
	fmt.Println(jsonBody)

	storeRefreshToken(jsonBody)
	return jsonBody.AccessToken
}

func storeRefreshToken(fitbitTokenResponse FitbitTokenResponse) {
	ioutil.WriteFile("refreshToken.txt", []byte(fitbitTokenResponse.RefreshToken), 0777)
}

type FitbitTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"`
}

type FitbitHeartIntradayData struct {
	Time  string
	Value int
}

type FitbitActivitiesHeartIntraday struct {
	Dataset []FitbitHeartIntradayData
}

type FitbitHeartRateResponse struct {
	ActivitiesHeartIntraday FitbitActivitiesHeartIntraday `json:"activities-heart-intraday"`
}

package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type BroadCastParams struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	ScheduledStartTime string `json:"scheduledStartTime"`
	PrivacyStatus      string `json:"privacyStatus"`
	LatencyPreference  string `json:"latencyPreference"`
	Thumbnail          string `json:"thumbnail"`
}

type RequestData struct {
	Title         string `json:"title"`
	Datetime      string `json:"datetime"`
	PrivacyStatus int    `json:"privacyStatus"`
	Latency       int    `json:"latency"`
	Description   string `json:"description"`
	Image         string `json:"image"`
}

func main() {
	// load env
	// if err := godotenv.Load(); err != nil {
	// 	log.Fatalf("Error loading .env file")
	// }
	e := echo.New()
	// allow cors settings from localhost:3000
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{os.Getenv("CLIENT_URL")},
	}))

	e.GET("/", func(c echo.Context) error {
		url, err := Auth()
		if err != nil {
			c.Logger().Error(err)
		}
		return c.Redirect(http.StatusFound, url)
	})

	// e.GET("/auth", func(c echo.Context) error {
	// 	// get path parameters
	// 	code := c.QueryParam("code")
	// 	log.Println(code)

	// 	// get token
	// 	conf := NewGoogleAuthConf()
	// 	ctx := c.Request().Context()
	// 	token, err := conf.Exchange(ctx, code)
	// 	if err != nil {
	// 		c.Logger().Error(err)
	// 	}
	// 	log.Println(token.AccessToken)
	// 	log.Println(token.RefreshToken)

	// 	return c.String(http.StatusOK, "Hello, World!")
	// })
	e.POST("/broadcasting", func(c echo.Context) error {
		// print log request
		// log.Println(c.Request())
		// read body as json
		body := make(map[string]interface{})
		err := c.Bind(&body)
		if err != nil {
			c.Logger().Error(err)
		}
		// log.Println(body)
		userID :=c.Request().Header.Get("X-Forwarded-User")
		if userID == "" || userID =="-" {
			// return forbidden error
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
		}

		// parse the request
		var requestData RequestData
		err = json.Unmarshal(dataJSON, &requestData)
		if err != nil {
			c.Logger().Error(err)
		}

		var broadCastParams BroadCastParams
		broadCastParams.Title = requestData.Title
		broadCastParams.Description = requestData.Description
		broadCastParams.ScheduledStartTime = requestData.Datetime
		switch requestData.PrivacyStatus {
		case 1:
			broadCastParams.PrivacyStatus = "public"
		case 2:
			broadCastParams.PrivacyStatus = "unlisted"
		}
		switch requestData.Latency {
		case 0:
			broadCastParams.LatencyPreference = "ultraLow"
		case 1:
			broadCastParams.LatencyPreference = "low"
		case 2:
			broadCastParams.LatencyPreference = "normal"
		}
		broadCastParams.Thumbnail = requestData.Image

		// get token
		token, err := getToken(c)
		if err != nil {
			c.Logger().Error(err)
		}

		// create youtube data client
		youtubeDataClient, err := newYouTubeDataClient(c, token)
		if err != nil {
			c.Logger().Error(err)
		}

		// create broadcasting
		broadcastId, err := createBroadcasting(youtubeDataClient, broadCastParams)
		if err != nil {
			c.Logger().Error(err)
		}

		// create stream
		streamId, err := createStream(youtubeDataClient, broadcastId)
		if err != nil {
			c.Logger().Error(err)
		}

		// bind stream
		err = bindStream(youtubeDataClient, broadcastId, streamId)
		if err != nil {
			c.Logger().Error(err)
		}

		// get stream info
		streamName, streamAddress, err := getStreamInfo(youtubeDataClient, streamId)
		if err != nil {
			c.Logger().Error(err)
		}

		// set thumbnail
		err = setThumbnail(youtubeDataClient, broadcastId, requestData.Image)
		if err != nil {
			c.Logger().Error(err)
		}

		// response
		response := map[string]string{
			"title":         requestData.Title,
			"videoId":       broadcastId,
			"streamName":    streamName,
			"streamAddress": streamAddress,
		}
		return c.JSON(http.StatusOK, response)
	})

	e.Logger.Fatal(e.Start(":8080"))
}

func NewGoogleAuthConf() *oauth2.Config {
	// // read config from json
	// credentialsJSON, err := os.ReadFile("client_secret_292739497457-q32m8gttceslc5e5vpfivsvcjgu0fq8h.apps.googleusercontent.com.json")
	// if err != nil {
	// 	log.Fatalf("Unable to read client secret file: %v", err)
	// }

	// // 第2引数に認証を求めるスコープを設定します.
	// config, err := google.ConfigFromJSON(credentialsJSON, youtube.YoutubeScope, youtube.YoutubeForceSslScope)
	// if err != nil {
	// 	log.Fatalf("Unable to parse client secret file to config: %v", err)
	// }

	// read client id and secret from environment variable
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	create oauth2 config
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://accounts.google.com/o/oauth2/auth",

			TokenURL: "https://oauth2.googleapis.com/token",
		},
		Scopes: []string{
			"https://www.googleapis.com/auth/youtube",
			"https://www.googleapis.com/auth/youtube.force-ssl",
		},
	}

	return config
}

func Auth() (string, error) {
	conf := NewGoogleAuthConf()
	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	log.Printf("Go to the following link in your browser then type the authorization code: %v", url)

	return url, nil
}

func getToken(c echo.Context) (string, error) {
	// read refresh token from environment variable
	refreshToken := os.Getenv("REFRESH_TOKEN")
	// tokenを更新する
	conf := NewGoogleAuthConf()
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	ctx := c.Request().Context()
	tokenSource := conf.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return "", err
	}
	return newToken.AccessToken, nil
}

func newYouTubeDataClient(c echo.Context, token string) (*youtube.Service, error) {
	ctx := c.Request().Context()
	config := NewGoogleAuthConf()
	youtubeService, err := youtube.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, &oauth2.Token{AccessToken: token})))
	if err != nil {
		return nil, err
	}
	return youtubeService, nil
}

func createBroadcasting(youtubeDataClient *youtube.Service, broadCastParams BroadCastParams) (string, error) {
	// create broadcast
	broadcast := &youtube.LiveBroadcast{
		Snippet: &youtube.LiveBroadcastSnippet{
			Title:              broadCastParams.Title,
			Description:        broadCastParams.Description,
			ScheduledStartTime: broadCastParams.ScheduledStartTime,
		},
		Status: &youtube.LiveBroadcastStatus{
			PrivacyStatus: broadCastParams.PrivacyStatus,
		},
		ContentDetails: &youtube.LiveBroadcastContentDetails{
			EnableDvr:         true,
			LatencyPreference: broadCastParams.LatencyPreference,
			EnableAutoStart:   true,
			EnableAutoStop:    true,
		},
	}
	call := youtubeDataClient.LiveBroadcasts.Insert([]string{"snippet,status,contentDetails"}, broadcast)
	response, err := call.Do()
	if err != nil {
		return "", err
	}
	log.Println("LiveBroadcast")
	log.Println(response.Id)
	return response.Id, nil
}

func createStream(youtubeDataClient *youtube.Service, broadCastId string) (string, error) {
	stream := &youtube.LiveStream{
		Snippet: &youtube.LiveStreamSnippet{
			Title: "test",
		},
		Cdn: &youtube.CdnSettings{
			IngestionType: "rtmp",
			Resolution:    "variable",
			FrameRate:     "variable",
		},
	}
	call := youtubeDataClient.LiveStreams.Insert([]string{"snippet,cdn"}, stream)
	response, err := call.Do()
	if err != nil {
		return "", err
	}
	log.Println("LiveStream")
	log.Println(response.Id)
	return response.Id, nil
}

func bindStream(youtubeDataClient *youtube.Service, broadCastId string, streamId string) error {
	call := youtubeDataClient.LiveBroadcasts.Bind(broadCastId, []string{"id", "snippet", "contentDetails", "status"})
	call.StreamId(streamId)
	_, err := call.Do()
	if err != nil {
		return err
	}
	return nil
}

func getStreamInfo(youtubeDataClient *youtube.Service, streamId string) (string, string, error) {
	call := youtubeDataClient.LiveStreams.List([]string{"snippet,cdn"})
	call.Id(streamId)
	response, err := call.Do()
	if err != nil {
		return "", "", err
	}
	log.Println("LiveStream")
	log.Println(response.Items[0].Cdn.IngestionInfo.StreamName)
	log.Println(response.Items[0].Cdn.IngestionInfo.IngestionAddress)
	return response.Items[0].Cdn.IngestionInfo.StreamName, response.Items[0].Cdn.IngestionInfo.IngestionAddress, nil
}

func setThumbnail(youtubeDataClient *youtube.Service, broadCastId string, thumbnail string) error {
	call := youtubeDataClient.Thumbnails.Set(broadCastId)
	// thumbnail is base64 encoded startwith data:image/png;base64,
	// so we need to remove the prefix split by comma
	decodedImage, err := base64.StdEncoding.DecodeString(thumbnail[strings.IndexByte(thumbnail, ',')+1:])
	if err != nil {
		return err
	}
	call.Media(bytes.NewReader(decodedImage)) // Adding bytes.NewReader to convert []byte to io.Reader
	_, err2 := call.Do()
	if err2 != nil {
		return err2
	}
	return nil
}

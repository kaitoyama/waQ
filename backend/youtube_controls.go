package main

import (
	"context"
	"fmt"

	"google.golang.org/api/youtube/v3"
)

type googleYouTubeClient struct{ service *youtube.Service }

func newGoogleYouTubeClient(ctx context.Context) (YouTubeClient, error) {
	token, err := getToken(ctx)
	if err != nil {
		return nil, err
	}
	service, err := newYouTubeDataClient(ctx, token)
	if err != nil {
		return nil, err
	}
	return &googleYouTubeClient{service: service}, nil
}

func (c *googleYouTubeClient) List(ctx context.Context) ([]Broadcast, error) {
	broadcasts := []Broadcast{}
	for _, status := range []string{"active", "upcoming"} {
		call := c.service.LiveBroadcasts.List([]string{"id,snippet,status,contentDetails"}).
			BroadcastStatus(status).
			MaxResults(50).
			Context(ctx)
		if err := call.Pages(ctx, func(response *youtube.LiveBroadcastListResponse) error {
			for _, item := range response.Items {
				if item.Status == nil || !controllableLifecycle(item.Status.LifeCycleStatus) {
					continue
				}
				broadcast, err := c.broadcast(ctx, item)
				if err != nil {
					return err
				}
				broadcasts = append(broadcasts, broadcast)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return broadcasts, nil
}

func controllableLifecycle(status string) bool {
	switch status {
	case "created", "ready", "testStarting", "testing", "liveStarting", "live":
		return true
	default:
		return false
	}
}

func (c *googleYouTubeClient) Get(ctx context.Context, id string) (Broadcast, error) {
	response, err := c.service.LiveBroadcasts.List([]string{"id,snippet,status,contentDetails"}).Id(id).Context(ctx).Do()
	if err != nil {
		return Broadcast{}, err
	}
	if len(response.Items) != 1 {
		return Broadcast{}, fmt.Errorf("broadcast not found")
	}
	return c.broadcast(ctx, response.Items[0])
}
func (c *googleYouTubeClient) Transition(ctx context.Context, id, target string) error {
	_, err := c.service.LiveBroadcasts.Transition(target, id, []string{"status"}).Context(ctx).Do()
	return err
}
func (c *googleYouTubeClient) broadcast(ctx context.Context, item *youtube.LiveBroadcast) (Broadcast, error) {
	broadcast := Broadcast{ID: item.Id, Title: item.Snippet.Title, URL: "https://www.youtube.com/watch?v=" + item.Id, Status: item.Status.LifeCycleStatus}
	if item.ContentDetails == nil || item.ContentDetails.BoundStreamId == "" {
		return broadcast, nil
	}
	streams, err := c.service.LiveStreams.List([]string{"status"}).Id(item.ContentDetails.BoundStreamId).Context(ctx).Do()
	if err != nil {
		return Broadcast{}, err
	}
	if len(streams.Items) == 1 && streams.Items[0].Status != nil {
		broadcast.StreamStatus = streams.Items[0].Status.StreamStatus
	}
	return broadcast, nil
}

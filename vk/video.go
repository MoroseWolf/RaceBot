package vk

import (
	"github.com/SevereCloud/vksdk/v3/api"
	"github.com/SevereCloud/vksdk/v3/object"
)

type MyVk struct {
	*api.VK
}
type MyVideo struct {
	object.VideoVideo
	LiveStartTime int    `json:"live_start_time"`
	LiveStatus    string `json:"live_status"`
}

type MyVideoGetResponse struct {
	Count int       `json:"count"`
	Items []MyVideo `json:"items"`
}

func (vk *MyVk) VideoGet(params api.Params) (response MyVideoGetResponse, err error) {
	err = vk.RequestUnmarshal("video.get", &response, params, api.Params{"extended": false})

	return
}

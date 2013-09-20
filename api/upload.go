package api

import (
	"fmt"
	"github.com/deet/picturelife-experimental-uploader/util"
	"log"
	"net/url"
)

type PendingMedia struct {
	CreatedAt          int64         `json:"create_at"`
	Error              bool          `json:"error"`
	ErrorData          []interface{} `json:"error_data"`
	Id                 string        `json:"id"`
	MediaType          string        `json:"media_type"`
	ProcessingComplete bool          `json:"processing_complete"`
	Status             string        `json:"status"`
	UpdatedAt          int64         `json:"updated_at"`
	UploadComplete     bool          `json:"upload_complete"`
	UserId             string        `json:"user_id"`
}

type Media struct {
	Bucket_Id          int
	Caption            string
	Color_Bands        string
	Comments_Count     int
	Created_At         int
	Deleted            bool
	Effective_Rating   float64
	Event_Id           string
	Face_Count         int
	Filesize           int
	Format             string
	Height             int
	Hidden             bool
	Id                 string
	Is_Best_Media      bool
	Likes_Count        int
	Media_Type         string
	Orientation        int
	Predicted_Rating   float64
	Privacy            int
	Processed          bool
	Shared_With_Family bool
	Taken_At           int
	//Temporary_Urls     TemporaryUrls
	Time_Zone        string
	Time_Zone_Offset int
	Updated_At       int
	Url              string
	User_Id          string
	Version          int
	Visible          bool
	Width            int
}

type MediasCreateResponse struct {
	ApiResponse
	PendingMedia PendingMedia `json:"pending_media"`
	Media        Media        `json:"media"`
}

type NewMedia struct {
	Signature  string
	S3Location string
	LocalPath  string
}

type SignatureResponse struct {
	MediaId string `json:"media_id"`
	Deleted bool   `json:"deleted"`
	Count   int
}

type CheckSignatureReponse struct {
	ApiResponse
	Signatures map[string]SignatureResponse
}

func (api *API) createMedia(newMedia NewMedia, force bool) (pendingMediaId, mediaId string, err error) {
	params := url.Values{}
	params.Add("signature", newMedia.Signature)
	params.Add("url", newMedia.S3Location)
	params.Add("local_path", newMedia.LocalPath)
	if force {
		params.Add("force", "true")
	}

	response := new(MediasCreateResponse)
	api.CallAndParseIntoWithOutput("medias/create", params, response, false)

	mediaId = response.Media.Id
	pendingMediaId = response.PendingMedia.Id
	return
}

func (api *API) Upload(filePath, sig string) (pendingMediaId, mediaId string, deleted bool, err error) {
	return api.UploadForce(filePath, sig, false)
}

func (api *API) UploadForce(filePath, sig string, force bool) (pendingMediaId, mediaId string, deleted bool, err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Upload error:", err)
		}
	}()

	var newMedia NewMedia
	newMedia.LocalPath = filePath

	if sig == "" {
		newMedia.Signature = util.CalculateSignature(filePath)
	} else {
		newMedia.Signature = sig
	}

	existingSignatures, err := api.CheckSignature(newMedia.Signature)
	//log.Println("check sig done")
	if err != nil {
		panic("Could not check signature")
	}
	// If there's an existing, non-deleted media there's no need to upload
	existingDeleted := false
	for _, sigResponse := range existingSignatures {
		mediaId = sigResponse.MediaId
		if sigResponse.MediaId == "" {
			continue
		}
		if sigResponse.Deleted == false {

			log.Println("Media exists already", mediaId, newMedia.Signature)
			return
		}
		existingDeleted = true
		if existingDeleted && !force {
			mediaId = sigResponse.MediaId
			deleted = existingDeleted
			log.Println("Existing deleted not forcing")
			return
		}
	}
	//log.Println("passed sig check")

	restartRulerUpload := force
	newMedia.S3Location, _, err = api.rulerUpload(filePath, sig, restartRulerUpload)
	if newMedia.S3Location == "" {
		panic("Ruler upload failed (location missing)")
	}
	if err != nil {
		panic(err)
	}

	//log.Println("Creating media from RULER upload.")

	pendingMediaId, mediaId, err = api.createMedia(newMedia, force)
	if err != nil {
		panic(fmt.Sprintln("medias/create error", err))
	}

	if mediaId != "" {
		log.Println("Media already uploaded.")
		log.Println("Media ID:", mediaId)
		return
	}

	//log.Println("Upload successful.")
	//log.Println("Pending media ID:", pendingMediaId)

	return
}

func (api *API) CheckSignature(sig string) (responses map[string]SignatureResponse, err error) {
	params := url.Values{}
	params.Add("signatures", sig) // param can actually be send as CSV list

	response := new(CheckSignatureReponse)
	api.CallAndParseIntoWithOutput("medias/check_signatures", params, response, false)

	//log.Println("done with call")

	responses = response.Signatures

	return
}

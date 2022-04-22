package speechtotext

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"

	speech "cloud.google.com/go/speech/apiv1"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

type GoogleCredentials struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

// BuildSpeechToTextClient create the speech client from the Google credentials
func BuildSpeechToTextClient(googleCredentials GoogleCredentials, context context.Context) *speech.Client {

	bytes, err := json.Marshal(googleCredentials)
	if err != nil {
		log.Println("Error initializing app:", err)
	}

	var scopes = []string{
		"https://www.googleapis.com/auth/cloud-platform",
	}

	cred, err := google.CredentialsFromJSON(context, bytes, scopes...)
	if err != nil {
		log.Println("Error initializing app:", err)
	}
	opt := option.WithCredentials(cred)

	speechClient, err := speech.NewClient(context, opt)

	if err != nil {
		log.Println("Error creating speech client")
	}

	return speechClient

}

// DonwloadsMediaVoice downloads the Voice file from the specified URL
func DonwloadsMediaVoice(urlFile, fileId string) string {
	response, err := http.Get(urlFile)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	fileTemp, errInputFile := ioutil.TempFile("", fmt.Sprintf("%s-*.mp4", fileId))
	if errInputFile != nil {
		log.Println("error creating temporary input file: ", err)
	}

	defer fileTemp.Close()

	_, err = io.Copy(fileTemp, response.Body)
	if err != nil {
		panic(err)
	}

	return fileTemp.Name()
}

// ConvertToFlac converts the file to FLAC audio file
func ConvertToFlac(inputFile, fileId string) []byte {
	outputTempFile, errOutputFile := ioutil.TempFile("", fmt.Sprintf("%s-*.flac", fileId))

	if errOutputFile != nil {
		log.Println("Error creating temporary output file: ", errOutputFile)
	}

	args := []string{
		"ffmpeg",
		"-i", inputFile,
		"-c:a",
		"flac",
		"-y", outputTempFile.Name(),
	}

	cmd := exec.Command(args[0], args[1:]...)

	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running ffmpeg, check if you have ffmpeg tool installed: %v", err)
	}

	flacFile, err := ioutil.ReadFile(outputTempFile.Name())

	if err != nil {
		log.Printf("Error reading flac file %s, error: %v", outputTempFile.Name(), err)
	}

	return flacFile
}

// TranscriptAudio converts the audio file to text
func TranscriptAudio(speechClient *speech.Client, context context.Context, urlFile, fileId, lang string) []string {

	temporaryFilePath := DonwloadsMediaVoice(urlFile, fileId)
	flacFile := ConvertToFlac(temporaryFilePath, fileId)
	language := lang

	resp, err := speechClient.Recognize(context, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:          speechpb.RecognitionConfig_FLAC,
			SampleRateHertz:   44100,
			LanguageCode:      language,
			AudioChannelCount: 2,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: flacFile},
		},
	})
	if err != nil {
		log.Printf("failed to recognize: %v", err)
	}
	var transcribedText []string

	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			transcribedText = append(transcribedText, alt.Transcript)
		}
	}

	return transcribedText
}

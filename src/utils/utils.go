package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"log"
	"math/rand"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rwcarlsen/goexif/exif"
)

const FEET_IN_METER = 3.28084
const METER_IN_MILE = 0.000621371

type Post struct {
	ImgBuf      bytes.Buffer
	Description string
}

type ImageMeta struct {
	Loc   string     `json:"loc"`
	Orig  [2]float64 `json:"orig"`
	Dest  []float64  `json:"dest"`
	Crow  float64    `json:"crow"`
	Walk  float64    `json:"walk"`
	Ratio float64    `json:"ratio"`
}

func getImagePaths(client *s3.Client) []string {

	filePaths := []string{}
	var continuationToken *string

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            aws.String(os.Getenv("AWS_BUCK")),
			ContinuationToken: continuationToken,
		}

		result, err := client.ListObjectsV2(context.TODO(), input)
		if err != nil {
			log.Fatal(err)
		}

		for _, obj := range result.Contents {
			filePaths = append(filePaths, *obj.Key)
		}

		if result.NextContinuationToken == nil {
			break // No more pages
		}

		continuationToken = result.NextContinuationToken
	}

	log.Printf("> images found = %d", len(filePaths))
	return filePaths
}

func downloadImage(client *s3.Client, imgKey string) []byte {

	downloader := manager.NewDownloader(client)
	headObjectOutput, err := client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(os.Getenv("AWS_BUCK")),
		Key:    aws.String(imgKey),
	})

	if err != nil {
		log.Fatalf("unable to get object size: %v", err)
	}

	// download image into byte array ([]byte)
	buf := make([]byte, *headObjectOutput.ContentLength)
	_, err = downloader.Download(
		context.TODO(),
		manager.NewWriteAtBuffer(buf),
		&s3.GetObjectInput{
			Bucket: aws.String(os.Getenv("AWS_BUCK")),
			Key:    aws.String(imgKey),
		},
	)

	if err != nil {
		log.Fatalf("unable to download: %v", err)
	}

	return buf
}

func extractImageMeta(imgBytes []byte) ImageMeta {

	x, err := exif.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		log.Fatal("Error decoding EXIF data:", err)
	}

	data, err := x.Get(exif.ImageDescription)
	if err != nil {
		log.Fatal("Error getting ImageDescription:", err)
	}

	jsonString := data.String()
	jsonString = jsonString[1 : len(jsonString)-1]

	var imgMeta ImageMeta
	err = json.Unmarshal([]byte(jsonString), &imgMeta)
	if err != nil {
		log.Fatalf("Error unmarshalling image metadata into json", err)
	}

	return imgMeta
}

func createPostDescription(imageMeta ImageMeta) string {
	d := imageMeta.Loc + "\n"
	d += fmt.Sprintf("It could be: %0.0f feet\n", imageMeta.Crow*FEET_IN_METER)
	d += fmt.Sprintf("But it's actually: %0.1f miles\n", imageMeta.Walk*METER_IN_MILE)
	d += fmt.Sprintf("Ratio: %0.2fx\n", imageMeta.Ratio)
	return d
}

func CreatePost() Post {

	// initialize s3 client
	cfg, _ := config.LoadDefaultConfig(
		context.TODO(),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				os.Getenv("AWS_USER"),
				os.Getenv("AWS_PSWD"),
				"",
			),
		),
	)
	cfg.Region = os.Getenv("AWS_REGN")
	client := s3.NewFromConfig(cfg)

	// get keys for all image candidates
	imgKeys := getImagePaths(client)

	// select random image for posting
	imgKey := imgKeys[rand.Intn(len(imgKeys))]
	log.Printf("> randomly selected image: %s\n", imgKey)

	// download image
	imgBytes := downloadImage(client, imgKey)

	// extract image description from metadata
	var post Post
	post.Description = createPostDescription(extractImageMeta(imgBytes))

	// munge image bytes into an actual jpeg
	img, err := jpeg.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		log.Fatalf("Unable to decode image: %v", err)
	}
	err = jpeg.Encode(&post.ImgBuf, img, nil)
	if err != nil {
		log.Fatalf("Unable to encode image: %v", err)
	}

	return post
}

package x

import (
	"context"
	"log"
	"os"
	"walkable/src/utils"

	// twitter api v1
	"github.com/dghubble/oauth1"
	"github.com/drswork/go-twitter/twitter"

	// twitter api v2
	"github.com/michimani/gotwi"
	"github.com/michimani/gotwi/tweet/managetweet"
	gotwiTypes "github.com/michimani/gotwi/tweet/managetweet/types"
)

func Run(post utils.Post) {

	// v1 api setup
	config := oauth1.NewConfig(os.Getenv("GOTWI_API_KEY"), os.Getenv("GOTWI_API_KEY_SECRET"))
	token := oauth1.NewToken(os.Getenv("TWIT_AT"), os.Getenv("TWIT_AS"))
	httpClient := config.Client(oauth1.NoContext, token)

	// v1 api - upload image
	clientV1 := twitter.NewClient(httpClient)
	media, _, _ := clientV1.Media.Upload(post.ImgBuf.Bytes(), "tweet_image")

	// v2 api - tweet with image
	in := &gotwi.NewClientInput{
		AuthenticationMethod: gotwi.AuthenMethodOAuth1UserContext,
		OAuthToken:           os.Getenv("TWIT_AT"),
		OAuthTokenSecret:     os.Getenv("TWIT_AS"),
	}
	clientV2, _ := gotwi.NewClient(in)

	// constuct new tweet
	m := &gotwiTypes.CreateInputMedia{MediaIDs: []string{media.MediaIDString}, TaggedUserID: nil}
	p := &gotwiTypes.CreateInput{
		Text:  gotwi.String(post.Description),
		Media: m,
	}

	// post to twitter
	output, err := managetweet.Create(context.Background(), clientV2, p)
	if err != nil {
		log.Println("> Error posting tweet:", err)
	}

	log.Println("> Tweet ID:", output.Data.ID)
}

package Core

//go get github.com/olivere/elastic/v7
import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	shared "httpRequestName/Shared"
	"net/http"
	"strconv"
	"time"
)

func GetESClient() (*elastic.Client, error) {
	//HTTPS yayinlarda TLS Certificat'i ihmal ediyor. Sadece Dev ortami icin tavsiye edilir.
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	elasticUrl, _ := Decrypt(shared.Config.ELASTICURL, shared.Config.SECRETKEY)
	elasticPassword, _ := Decrypt(shared.Config.ELASTICPASSWORD, shared.Config.SECRETKEY)
	elasticUser, _ := Decrypt(shared.Config.ELASTICUSER, shared.Config.SECRETKEY)
	client, err := elastic.NewClient(elastic.SetURL(elasticUrl),
		elastic.SetHttpClient(httpClient),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false), elastic.SetBasicAuth(elasticUser, elasticPassword))

	fmt.Println("ES initialized...")

	return client, err
}

func InsertLog(errorLog interface{}, indexName string) error {
	ctx := context.Background()
	esclient, err := GetESClient()
	if err != nil {
		fmt.Println("Error initializing : ", err)
		return err
	}
	dataJSON, err := json.Marshal(errorLog)
	js := string(dataJSON)

	//Check Index Exist Or Not
	exists, err := esclient.IndexExists(indexName).Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	//If not Exist Alias and Sharding Mapping Added.
	if !exists {
		var mapping = ElasticMaps[indexName]
		fmt.Println(mapping)
		//Set RandomName for new ElasticIndex
		randomindexName := strconv.FormatInt(makeTimestampMilli(), 10)
		randomindexName = indexName + randomindexName
		_, err = esclient.CreateIndex(randomindexName).Body(mapping).Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
		_, err = esclient.Index().Index(indexName).BodyJson(js).Do(ctx)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = esclient.Index().Index(indexName).BodyJson(js).Do(ctx)
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("[Elastic][Insert-%s]Insertion Successful", indexName)
	return nil
}

func unixMilli(t time.Time) int64 {
	return t.Round(time.Millisecond).UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func makeTimestampMilli() int64 {
	return unixMilli(time.Now())
}

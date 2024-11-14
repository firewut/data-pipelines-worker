# Data pipelines worker
This is unified project for all workers

## Configuration
Main Configuration `./config/config.yaml` ( example included to sources )

Storage configuration `./config/storage_credentials.json`. Uses MINIO exported credentials
```
    {
        "accessKey": "----",
        "api": "s3v4",
        "path": "auto",
        "secretKey": "----",
        "url": "localhost:9000",
        "bucket": "data-pipelines"
    }
```

## Start
Just execute following command in terminal and it should be up and running
```
make start
```

curl -X POST -H "Content-Type: application/json" -d '{"pipeline":{"slug":"openai-yt-short-generation"},"block":{"slug":"get-event-text", "input": {"user_prompt": "What happened years ago today October twenty fourth?"}}}' "http://192.168.1.116:8080/pipelines/openai-yt-short-generation/start"

## Resume
curl -X POST -H "Content-Type: application/json" -d '{"pipeline":{"slug":"openai-yt-short-generation", "processing_id":"99e4d0d9-eaf0-4dea-89dd-15b5cbb5ce1f"},"block":{"slug":"send-event-images-moderation-to-telegram" }}' "http://192.168.1.116:8080/pipelines/openai-yt-short-generation/resume"

curl -X POST -H "Content-Type: application/json" -d '{"pipeline":{"slug":"openai-yt-short-generation", "processing_id":"0e2796da-d262-4e2c-b9f0-bf792de1f0dc"},"block":{"slug":"get-event-images"}}' "http://localhost:8080/pipelines/openai-yt-short-generation/resume"


curl -X POST "http://192.168.1.116:8080/pipelines/openai-yt-short-generation/resume" \
  -H "Content-Type: multipart/form-data" \
  -F "pipeline.slug=openai-yt-short-generation" \
  -F "pipeline.processing_id=14c9f824-2211-45f2-9378-c875b3e1e34c" \
  -F "block.slug=get-event-images" \
  -F "block.target_index=1" \
  -F "block.input.video=@/path/to/video.mp4"

For arrays use:
```
-F "block.input.items[]=3" \
-F "block.input.items[]=2" \
```
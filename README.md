# Data pipelines worker
This project is a pipeline management system designed to process and transform data through modular "blocks." It focuses on generating multimedia content, such as videos, images, and text, using AI integrations while incorporating moderation workflows through social network platforms. The system allows flexible inter-block communication, parallel processing, and seamless recovery mechanisms. It's also tailored for scalability and autonomy, with plans for deployment on resource-constrained devices.

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

curl -X POST -H "Content-Type: application/json" -d '{"pipeline":{"slug":"openai-motivational-quote-to-video"},"block":{"slug":"analyze-user-input", "input": {"user_prompt": "Your time is limited, so do not waste it living someone else life"}}}' "http://192.168.1.116:8080/pipelines/openai-motivational-quote-to-video/start"


## Resume
curl -X POST -H "Content-Type: application/json" -d '{"pipeline":{"slug":"openai-yt-short-generation", "processing_id":"99e4d0d9-eaf0-4dea-89dd-15b5cbb5ce1f"},"block":{"slug":"send-event-images-moderation-to-telegram" }}' "http://192.168.1.116:8080/pipelines/openai-yt-short-generation/resume"

curl -X POST -H "Content-Type: application/json" -d '{"pipeline":{"slug":"openai-yt-short-generation", "processing_id":"0e2796da-d262-4e2c-b9f0-bf792de1f0dc"},"block":{"slug":"get-event-images"}}' "http://localhost:8080/pipelines/openai-yt-short-generation/resume"

curl -X POST -H "Content-Type: application/json" -d '{"pipeline":{"slug":"openai-motivational-quote-to-video", "processing_id":"baceaadc-b8d8-4576-998a-923737044fcf"},"block":{"slug":"add-text-to-event-images"}}' "http://192.168.1.116:8080/pipelines/openai-motivational-quote-to-video/resume"

curl -X POST -H "Content-Type: multipart/form-data" \
  -F "pipeline.slug=openai-podcast-summary" \
  -F "block.slug=get-event-transcription" \
  -F "block.input.audio_file=@../../output000.mp3" \
  "http://192.168.1.116:8080/pipelines/openai-podcast-summary/start"

curl -X POST -H "Content-Type: application/json" -d '{"pipeline":{"slug":"openai-podcast-summary", "processing_id":"43aa8a6a-9088-42c7-8ea9-773f10b9d5ea"},"block":{"slug":"get-summary-of-a-podcast"}}' "http://192.168.1.116:8080/pipelines/openai-podcast-summary/resume"


curl -X POST -H "Content-Type: multipart/form-data" \
  -F "pipeline.slug=openai-mux-subtitles-to-video" \
  -F "block.slug=upload-video-file" \
  -F "block.input.file=@../../video.mp4" \
  "http://192.168.1.116:8080/pipelines/openai-mux-subtitles-to-video/start"



For arrays use:
```
-F "block.input.items[]=3" \
-F "block.input.items[]=2" \
```
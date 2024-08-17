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
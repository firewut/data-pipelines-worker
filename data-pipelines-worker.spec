Name:           data-pipelines-worker
Version:        1.0.6
Release:        1%{?dist}
Summary:        Data pipelines worker
License:        Custom License
Group:          Development/Tools

%description
Pipelines management and execution system.

%prep

%build

%install
mkdir -p %{buildroot}/usr/local/bin
install -m 0755 %{_sourcedir}/data-pipelines-worker-linux %{buildroot}/usr/local/bin/data-pipelines-worker

mkdir -p %{buildroot}/etc/data-pipelines-worker/config
cp -r %{_sourcedir}/config/* %{buildroot}/etc/data-pipelines-worker/config/

echo '{"accessKey": "-","api": "s3v4","path": "auto","secretKey": "-","url": "localhost:9000","bucket": "data-pipelines"}' > %{buildroot}/etc/data-pipelines-worker/config/minio_storage_credentials.json

mkdir -p %{buildroot}/etc/systemd/system/
install -D -m 0644 %{_sourcedir}/etc/systemd/system/data-pipelines-worker.service %{buildroot}/etc/systemd/system/data-pipelines-worker.service

%post
systemctl enable data-pipelines-worker.service > /dev/null 2>&1 || :
systemctl start data-pipelines-worker.service > /dev/null 2>&1 || :

%preun
if [ $1 -eq 0 ]; then
    systemctl stop data-pipelines-worker.service > /dev/null 2>&1 || :
    systemctl disable data-pipelines-worker.service > /dev/null 2>&1 || :
fi

%files
/usr/local/bin/data-pipelines-worker
/etc/data-pipelines-worker/config/*
/etc/data-pipelines-worker/config/pipelines/*
%config(noreplace) /etc/data-pipelines-worker/config/minio_storage_credentials.json
%config(noreplace) /etc/data-pipelines-worker/config/openai_credentials.json
%config(noreplace) /etc/data-pipelines-worker/config/telegram_credentials.json
/etc/systemd/system/data-pipelines-worker.service

%changelog
* Sun Nov 17 2024 Andrei Chibisov <andrey844@gmail.com> - 1.0.6
- Fixed data races at Pipeline Processing Logs gathering
- Fixed bug with load data  when input passed
* Fri Nov 15 2024 Andrei Chibisov <andrey844@gmail.com> - 1.0.5
- Added HTTP Handler to GET specific Pipeline Processings
- Added HTTP Handler to GET specific Pipeline Processing
- Fixed an Issue with LocalStorage
* Thu Nov 14 2024 Andrei Chibisov <andrey844@gmail.com> - 1.0.4
- Added Pipeline Log to Storage dump
* Tue Nov 13 2024 Andrei Chibisov <andrey844@gmail.com> - 1.0.3
- Added HTTP Handler to GET specific Pipeline
* Mon Nov 11 2024 Andrei Chibisov <andrey844@gmail.com> - 1.0.2
- Added multipart/form-data API Interface
- Added block `audio_from_video`
- Added block `subtitles_from_transcription`
- Added pipeline to add generated subtitles file to uploaded video
* Fri Nov 08 2024 Andrei Chibisov <andrey844@gmail.com> - 1.0.1
- Added Swagger
* Mon Nov 04 2024 Andrei Chibisov <andrey844@gmail.com> - 1.0.0
- Initial RPM release.
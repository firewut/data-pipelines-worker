Name:           data-pipelines-worker
Version:        1.0.0
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

%files
/usr/local/bin/data-pipelines-worker
/etc/data-pipelines-worker/config/*
/etc/data-pipelines-worker/config/pipelines/*
/etc/data-pipelines-worker/config/minio_storage_credentials.json

%changelog
* Mon Nov 04 2024 Andrei Chibisov <andrey844@gmail.com> - 0.0.1
- Initial RPM release.
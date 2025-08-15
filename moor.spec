%define name moor
%define version 2.0.4
%define release 1%{?dist}

Summary:  Moar is a pager. It's designed to just do the right thing without any configuration.
Name:     %{name}
Version:  %{version}
Release:  %{release}
License:  MIT License
URL:      https://github.com/walles/moar
Source0:  https://github.com/walles/moar/archive/refs/tags/v%{version}.tar.gz

Provides: moar = %{version}-%{release}
Obsoletes: moar < %{version}-%{release}

%define debug_package %{nil}

BuildRequires: curl
BuildRequires: gcc
BuildRequires: make
BuildRequires: gzip
BuildRequires: golang
BuildRequires: upx
BuildRequires: git

%description
Moar is a pager. It reads and displays UTF-8 encoded text from files or pipelines.

%prep
%setup -q

%build
GO111MODULE=on CGO_ENABLED=0 go build -v -trimpath -modcacherw -tags netgo \
    -ldflags="-s -w -X main.versionString=%{version}" \
    -o %{name} ./cmd/%{name}
strip --strip-all %{name}
upx %{name}
gzip %{name}.1

%install
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_mandir}/man1
install -m 755 %{name} %{buildroot}%{_bindir}
install -m 644 %{name}.1.gz %{buildroot}%{_mandir}/man1

%files
%doc README.md
%license LICENSE
%{_bindir}/%{name}
%{_mandir}/man1/%{name}.1.gz

%changelog
* Fri Aug 15 2025 - Danie de Jager 2.0.4-1
* Tue Aug 12 2025 - Danie de Jager 2.0.3-1
* Mon Aug 11 2025 - Danie de Jager 2.0.1-1
- rename from moar to moor
* Tue Jul 29 2025 - Danie de Jager 1.33.0-1
* Thu Jul 24 2025 - Danie de Jager 1.32.6-1
* Tue Jul 22 2025 - Danie de Jager 1.32.5-1
* Mon Jul 21 2025 - Danie de Jager 1.32.4-1
* Tue Jul 8 2025 - Danie de Jager 1.32.3-1
* Mon Jun 30 2025 - Danie de Jager 1.32.2-1
* Thu Jun 26 2025 - Danie de Jager 1.32.0-1
* Wed Jun 11 2025 - Danie de Jager 1.31.10-1
* Wed Jun 11 2025 - Danie de Jager 1.31.9-1
* Thu May 29 2025 - Danie de Jager 1.31.8-1
* Tue May 20 2025 - Danie de Jager 1.31.7-1
* Sun May 18 2025 - Danie de Jager 1.31.6-1
* Fri Apr 25 2025 - Danie de Jager 1.31.5-1
* Tue Feb 25 2025 - Danie de Jager 1.31.3-1
* Mon Feb 17 2025 - Danie de Jager 1.31.2-2
* Mon Jan 13 2025 - Danie de Jager 1.31.2-1
* Fri Jan 10 2025 - Danie de Jager 1.31.1-1
* Fri Jan 3 2025 - Danie de Jager 1.30.1-1
* Thu Nov 28 2024 - Danie de Jager 1.30.0-1
* Mon Nov 18 2024 - Danie de Jager 1.29.0-1
* Thu Nov 7 2024 - Danie de Jager 1.28.2-1
- Remedy two race conditions.
* Sun Nov 3 2024 - Danie de Jager 1.28.1-1
* Thu Oct 31 2024 - Danie de Jager - 1.27.3-1
- Prevent blank last column on Windows
- Fish shell specific help text for setting moar as your pager
- PowerShell specific help text for setting moar as your pager
- Assume VGA color scheme on 16 color terminals
* Thu Oct 24 2024 - Danie de Jager - 1.27.2-1
- Handle wide chars in the input
* Mon Sep 16 2024 - Danie de Jager - 1.27.1-1
- Accept \ characters in URLs.
* Tue Sep 9 2024 - Danie de Jager - 1.27.0-1
* Tue Aug 27 2024 - Danie de Jager - 1.26.0-1
* Mon Aug 12 2024 - Danie de Jager - 1.25.4-1
* Tue Aug 6 2024 - Danie de Jager - 1.25.2-1
* Tue Jul 16 2024 - Danie de Jager - 1.25.1-1
* Sun Jul 14 2024 - Danie de Jager - 1.25.0-1
* Tue Jul 9 2024 - Danie de Jager - 1.24.6-1
* Tue May 21 2024 - Danie de Jager - 1.23.15-1
* Fri Mar 1 2024 - Danie de Jager - 1.23.6-1
- Initial RPM build

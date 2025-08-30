Name:    moor
Summary: Simple UTF-8 pager with sensible defaults
Version: 2.0.5
Release: 1%{?dist}
License: BSD-2-Clause
URL:     https://github.com/walles/moor
Source0: https://github.com/walles/moor/archive/refs/tags/v%{version}.tar.gz

%define debug_package %{nil}

BuildRequires: curl
BuildRequires: gcc
BuildRequires: make
BuildRequires: gzip
BuildRequires: golang
BuildRequires: git

%description
Moor is a pager for UTF-8 encoded text. It reads and displays
text from files or from pipelines. It is designed to work out of
the box with sensible defaults, without requiring user configuration.

%prep
%setup -q

%build
GO111MODULE=on go build -v -trimpath -modcacherw \
   -ldflags="-s -w -X main.versionString=%{version}" \
   -o %{name} ./cmd/%{name}

strip --strip-all %{name}
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
* Tue Aug 26 2025 - Danie de Jager <danie.dejager@gmail.com>- 2.0.5-1
- Fixed a crash related to intermittent problem related to scrolling around the switch from line numbers 999 to 1000.
- Mac keyboards can now press option-arrow to scroll sideways one column at a time.
* Sat Aug 23 2025 - Danie de Jager <danie.dejager@gmail.com>- 2.0.4-2
- Update license

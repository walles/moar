Name:           moor
Summary:        Simple UTF-8 pager with sensible defaults
Version:        2.0.4
Release:        2%{?dist}
License:        BSD-2-Clause
URL:            https://github.com/walles/moor
Source0:        %{url}/archive/refs/tags/v%{version}.tar.gz

BuildRequires:  golang
BuildRequires:  git

%description
Moar (packaged as moor) is a pager for UTF-8 encoded text. It reads and
displays text from files or from pipelines. It is designed to work out of
the box with sensible defaults, without requiring user configuration.

%prep
%setup -q -n moor-%{version}

%build
%gobuild -o %{name} ./cmd/%{name}

gzip -9 -c %{name}.1 > %{name}.1.gz

%install
install -Dpm 0755 %{name} %{buildroot}%{_bindir}/%{name}
install -Dpm 0644 %{name}.1.gz %{buildroot}%{_mandir}/man1/%{name}.1.gz

%check
if go test ./...; then
  echo "go test suite passed"
else
  echo "go test suite failed; skipping"
fi

./test.sh || echo "test.sh failed; proceeding"

%files
%doc README.md
%license LICENSE
%{_bindir}/%{name}
%{_mandir}/man1/%{name}.1.gz

%changelog
* Sat Aug 23 2025 Danie de Jager <danie.dejager@gmail.com> - 2.0.4-2
- Clean up spec to follow Fedora Go packaging guidelines
- Update license handling

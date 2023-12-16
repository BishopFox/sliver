FROM golang:bookworm as base
LABEL org.opencontainers.image.source https://github.com/rsteube/carapace
USER root

FROM base as bat
ARG version=0.24.0
RUN curl -L https://github.com/sharkdp/bat/releases/download/v${version}/bat-v${version}-x86_64-unknown-linux-gnu.tar.gz \
  | tar -C /usr/local/bin/ --strip-components=1  -xvz bat-v${version}-x86_64-unknown-linux-gnu/bat \
  && chmod +x /usr/local/bin/bat

FROM base as ble
RUN git clone --recursive https://github.com/akinomyoga/ble.sh.git \
 && apt-get update && apt-get install -y gawk \
 && make -C ble.sh

FROM base as elvish
ARG version=0.19.2
RUN curl https://dl.elv.sh/linux-amd64/elvish-v${version}.tar.gz | tar -xvz \
  && mv elvish-* /usr/local/bin/elvish

FROM base as goreleaser
ARG version=1.21.2
RUN curl -L https://github.com/goreleaser/goreleaser/releases/download/v${version}/goreleaser_Linux_x86_64.tar.gz | tar -xvz goreleaser \
  && mv goreleaser /usr/local/bin/goreleaser

FROM rsteube/ion-poc as ion-poc
#FROM rust as ion
#ARG version=master
#RUN git clone --single-branch --branch "${version}" --depth 1 https://gitlab.redox-os.org/redox-os/ion/ \
# && cd ion \
# && RUSTUP=0 make # By default RUSTUP equals 1, which is for developmental purposes \
# && sudo make install prefix=/usr \
# && sudo make update-shells prefix=/usr

FROM base as nushell
ARG version=0.85.0
RUN curl -L https://github.com/nushell/nushell/releases/download/${version}/nu-${version}-x86_64-unknown-linux-gnu.tar.gz | tar -xvz \
 && mv nu-${version}-x86_64-unknown-linux-gnu/nu* /usr/local/bin

FROM base as oil
ARG version=0.18.0
RUN apt-get update && apt-get install -y libreadline-dev
RUN curl https://www.oilshell.org/download/oil-${version}.tar.gz | tar -xvz \
  && cd oil-*/ \
  && ./configure \
  && make \
  && ./install

FROM base as starship
ARG version=1.16.0
RUN wget -qO- "https://github.com/starship/starship/releases/download/v${version}/starship-x86_64-unknown-linux-gnu.tar.gz" | tar -xvz starship \
 && mv starship /usr/local/bin/

FROM base as vivid
ARG version=0.9.0
RUN wget -qO- "https://github.com/sharkdp/vivid/releases/download/v${version}/vivid-v${version}-x86_64-unknown-linux-gnu.tar.gz" | tar -xvz vivid-v${version}-x86_64-unknown-linux-gnu/vivid \
 && mv vivid-v${version}-x86_64-unknown-linux-gnu/vivid /usr/local/bin/

FROM base as mdbook
ARG version=0.4.35
RUN apt-get update && apt-get install -y unzip \
  && curl -L "https://github.com/rust-lang/mdBook/releases/download/v${version}/mdbook-v${version}-x86_64-unknown-linux-gnu.tar.gz" | tar -xvz mdbook \
  && wget -q "https://github.com/Michael-F-Bryan/mdbook-linkcheck/releases/download/v0.7.7/mdbook-linkcheck.x86_64-unknown-linux-gnu.zip" \
  && unzip mdbook-linkcheck.x86_64-unknown-linux-gnu.zip mdbook-linkcheck \
  && chmod +x mdbook-linkcheck \
  && mv mdbook mdbook-linkcheck /usr/local/bin/

FROM base
RUN apt-get update && apt-get install -y libicu72
RUN wget -q  https://github.com/PowerShell/PowerShell/releases/download/v7.3.8/powershell_7.3.8-1.deb_amd64.deb\
  && dpkg -i powershell_7.3.8-1.deb_amd64.deb \
  && rm powershell_7.3.8-1.deb_amd64.deb

RUN apt-get update \
  && apt-get install -y fish \
  elvish \
  expect \
  shellcheck \
  tcsh \
  xonsh \
  zsh

RUN pwsh -Command "Install-Module PSScriptAnalyzer -Scope AllUsers -Force"

RUN git config --system safe.directory '*'

COPY --from=bat /usr/local/bin/* /usr/local/bin/
COPY --from=ble /go/ble.sh /opt/ble.sh
COPY --from=elvish /usr/local/bin/* /usr/local/bin/
COPY --from=goreleaser /usr/local/bin/* /usr/local/bin/
#COPY --from=ion /ion/target/release/ion /usr/local/bin/
COPY --from=ion-poc /usr/local/bin/ion /usr/local/bin/
COPY --from=nushell /usr/local/bin/* /usr/local/bin/
COPY --from=mdbook /usr/local/bin/* /usr/local/bin/
COPY --from=oil /usr/local/bin/* /usr/local/bin/
COPY --from=starship /usr/local/bin/* /usr/local/bin/
COPY --from=vivid /usr/local/bin/* /usr/local/bin/

ADD .dockerfile/root /root
ADD .dockerfile/usr/local/bin/* /usr/local/bin/

ENV TERM xterm
ENTRYPOINT [ "entrypoint.sh" ]

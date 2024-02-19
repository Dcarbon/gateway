FROM dcarbon/go-shared as builder

WORKDIR /dcarbon/gateway
COPY . .

RUN go mod tidy && go build -buildvcs=false  && \
    cp  gateway /usr/bin && \
    echo "Build image successs...!"


FROM dcarbon/dimg:minimal

COPY --from=builder /usr/bin/gateway /usr/bin/gateway
COPY --from=builder /dcarbon/arch-proto/swagger/api.swagger.json /etc/config/api.json

ENV DOC_FILE=/etc/config/api.json

CMD [ "gateway" ]
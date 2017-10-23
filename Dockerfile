FROM 1.9.1-alpine3.6
COPY tendereum /tendereum
ENTRYPOINT ["/tendereum"]
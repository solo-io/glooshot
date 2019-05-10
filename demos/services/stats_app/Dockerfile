# calls the go process only, making it the first PID
FROM alpine
WORKDIR /app
ADD _output/stats-app /app
EXPOSE 8080
ENTRYPOINT ./stats-app

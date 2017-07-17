## Lunch

lunch needs a better name

lunch is also a slack integration, written in go, to be used for voting on weekly lunch

Running
-----
```
docker build . -t <tag>
docker run -p 8765:8765 -e "SLACKKEY=<slack-key>" <tag>
```


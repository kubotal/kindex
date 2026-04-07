
```
helm -n kindex template --debug kindex1 ./helm/kindex >xx.yaml



helm -n kindex upgrade -i kindex1 ./helm/kindex --create-namespace

helm -n kindex uninstall kindex1 

```





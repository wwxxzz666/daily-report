$env:PATH = 'D:\anaconda3\pkgs\go-1.26.3-h5588389_0\go\bin;C:\Users\37119\go\bin;' + $env:PATH
$env:GOROOT = 'D:\anaconda3\pkgs\go-1.26.3-h5588389_0\go'
$env:GOPROXY = 'https://goproxy.cn,direct'
Set-Location 'd:\study\日报助手'
wails build 2>&1
Write-Host "EXIT CODE: $LASTEXITCODE"

# ec2-hosts

EC2インスタンスの `/etc/hosts` を更新するscriptです。

## install

```
go get -u github.com/yoppi/ec2-hosts
```

設定ファイルをバイナリに含めるため、ビルド時にgo-bindataを使用するため、合わせてインストールします。

```
 go get -u github.com/jteeuwen/go-bindata/...
```

## usage

### settings

cloneした `config/default.tml` ファイルを編集して、`config/ec2-hosts.tml` ファイルを作成します。プログラムがアクセスする設定ファイルのパスは固定になっています。

```
[aws]
region = "ap-northeast-1"
access_key_id = "SET YOUR ACCESS KEY ID"
secret_access_key = "SET YOUR SECRET ACCESS KEY"

[tags]
Name = "target-host"
```

上記の設定ファイルは、EC2インスタンスのtagの `Name` が `target-host` であるインスタンスのPrivate IPAddressを取得し、
/etc/hostsに

```
[PrivateIPAddress] target-host
```

として登録することを意味しています。
すでに `target-host` が/etc/hostsに登録されていた場合、取得したIPAddressで上書きします。
tagの名前に指定する値は配列も指定でき、

```
[tags]
Name = ["target-host", "target-host2"]
```

といった指定も可能です。

### build

makeを実行すると、作成した設定ファイルもバイナリに含まれ、 `ec2-hosts` という名前で実行ファイルを生成します。

```
$ make
```

ローカルで実行するとローカルの/etc/hostsが更新されます(root権限がなければ通常/etc/hostsは更新できないのでsudo等を付けて実行してください)

また、基本的にEC2インスタンス上で使うことを想定しているので、その場合は、

```
$ make build-release
```

ターゲットを指定してください。Linux用にクロスコンパイルします。
生成されたバイナリをEC2インスタンスの任意のパスに保存し、実行することで/etc/hostsを更新できるようになります。

## usage scene

/etc/rc.localなどで実行するようにしておくと、EC2インスタンス起動時に/etc/hostsを更新してくれるので、
マシン間の通信を特定の名前でアクセスできるようになり、運用を楽にできます。

例として、私達が運用している方法として、PackerでインスタンスをAMI化するときにこのバイナリを仕込んでおく、というものです。
/etc/rc.localで実行するようにしておくと、AMI起動時に/etc/hostsが更新されているので、設定ファイル等でホストの指定をIPAddressではなくホスト名を指定できるようになることから、ansibleなどによる設定ファイルのプロビジョニングが楽になると思っています。

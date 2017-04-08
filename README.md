# ec2-hosts

ec2-hosts is update `/etc/hosts` on EC2 instances.

## install

```
go get -u github.com/yoppi/ec2-hosts
```

## usage

### settings

`config/ec2-hosts.toml.orig` ファイルを編集して、`config/ec2-hosts.tml` ファイルを作成します。
ec2-hostsがアクセスする設定ファイルのパスは `/etc/ec2-hosts/ec2-hosts.toml` がデフォルトです。もしくは、 `-config` オプションで実行時に任意のパスを指定できます。

```
[aws]
region = "ap-northeast-1"
access_key_id = "SET YOUR ACCESS KEY ID"
secret_access_key = "SET YOUR SECRET ACCESS KEY"

[tags]
Name = "target-host"

[groupTags]
env = "production"
```

設定ファイルの一例です。それぞれのTable毎について説明します。

#### aws

AWSの設定項目です。ec2-hostsはEC2インスタンス情報を取得するため、EC2に対するREAD権限があるkey-pairを指定してください。

#### tags

上記の設定ファイルは、EC2インスタンスのtagの `Name` が `target-host` であるインスタンスのPrivate IPAddressを取得し、 `/etc/hosts` に

```
<PrivateIPAddress> target-host
```

として登録することを意味しています。
すでに `target-host` が/etc/hostsに登録されていた場合、取得したIPAddressで上書きします。
tagの名前に指定する値は配列も指定でき、

```
[tags]
Name = ["target-host", "target-host2"]
```

といった指定も可能です。

#### groupTags

上記の設定ファイルは、EC2インスタンスのtagのKeyが `env` で `production` であるインスタンスを取得し、それぞれのインスタンスのtagのkeyが `Name`
をホスト名としてPrivate IPAddressを `/etc/hosts` に登録します。
同様に、

```
[groupTags]
role = ["app", "worker"]
```

の様に値を配列で指定できます。

### build

実行ファイルはreleasesからダウンロード、もしくは手動でビルドします。

makeを実行すると、`ec2-hosts` という名前で実行ファイルを生成します。

```
$ make
```

また、基本的にEC2インスタンスのLinux上で使うことを想定しているので、その場合は、

```
$ make build-release
```

ターゲットを指定してください。GOOS=linux GOARCH=amd64でクロスコンパイルします。

### exec

生成されたバイナリをEC2インスタンスの任意のパスに保存し、設定ファイルを指定、もしくは、所定の位置において実行することで/etc/hostsを更新できるようになります。

```
$ ec2-hosts -config [/etc/ec2-hosts/ec2-hosts.toml]
```

実行時に `-loop` オプションを指定すると強制停止するまで実行し続けます。デーモン化する場合はsystemdなどを併用すると良いでしょう。

```
$ ec2-hosts -loop # unlimited loop
```

## use cases

/etc/rc.localやsystemd等で実行するようにしておくと、EC2インスタンス起動時に(-loopフラグを付けていると定期的に)/etc/hostsを更新してくれるので、EC2インスタンス間の通信をホスト名でアクセスできるようになり、運用を楽にできます。

私達が運用している方法として、EC2インスタンスをAMI化するときにこのバイナリを仕込んでおく、というものです。
systemdでデーモンとして実行するようにしておくと、AMIからインスタンス作成時に/etc/hostsが更新されているので、設定ファイル等でホストの指定をIPAddressではなくホスト名を指定できるようになることから、ansibleなどによる設定ファイルのプロビジョニングが楽になると思っています。

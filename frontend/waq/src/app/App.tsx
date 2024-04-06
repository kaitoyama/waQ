import React, { useState } from 'react';
import { Button, Card, Col, Input, Layout, message, notification, Radio, Row, Skeleton, Spin, Tooltip } from "antd";
import { DatePicker } from 'antd';
import { Typography } from "antd";
import { Form } from "antd";
import { Image, Upload } from 'antd';
import type { GetProp, UploadFile, UploadProps } from 'antd';
import { PlusOutlined } from '@ant-design/icons';

import hmacSHA512 from 'crypto-js/hmac-sha512';
import Base64 from 'crypto-js/enc-base64';
import imageCompression from 'browser-image-compression';
window.addEventListener('beforeunload', (e: BeforeUnloadEvent) => {
  e.preventDefault()
  return message
})

const { Title, Text, Link } = Typography;
const { Header, Content, Footer } = Layout;

const formItemLayout = {
  labelCol: {
    xs: { span: 24 },
    sm: { span: 6 },
  },
  wrapperCol: {
    xs: { span: 24 },
    sm: { span: 14 },
  },
};
type FileType = Parameters<GetProp<UploadProps, 'beforeUpload'>>[0];

const getBase64 = (file: FileType): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = (error) => reject(error);
  });


function App() {
  const [streamkey, setStreamKey] = useState('ストリームキー');
  const [videoID, setVideoID] = useState('');
  const [title, setTitle] = useState('');
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewImage, setPreviewImage] = useState('');
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [loading, setLoading] = useState(false);
  const handlePreview = async (file: UploadFile) => {
    if (!file.url && !file.preview) {
      const preview = await getBase64(file.originFileObj as FileType);
      file.preview = preview;
    }

    setPreviewImage(file.url || (file.preview as string));
    setPreviewOpen(true);
  };


  const handleChange: UploadProps['onChange'] = ({ fileList: newFileList }) =>
    setFileList(newFileList);

  const uploadButton = (
    <button style={{ border: 0, background: 'none' }} type="button">
      <PlusOutlined />
      <div style={{ marginTop: 8 }}>Upload</div>
    </button>
  );
  const onFinish = async (values: any) => {
    setLoading(true);
    console.log(values);
    // バックエンドに送信するデータの準備
    // 画像サイズを圧縮 上限は2MB
    const options = {
      maxSizeMB: 2,
      maxWidthOrHeight: 1920,
      useWebWorker: true
    };
    var requestData: { [x: string]: any; };
    if (fileList.length > 0) {
      const compressedFile = await imageCompression(fileList[0].originFileObj as File, options);

      requestData = {
        ...values, // フォームからの入力データを含める
        // 画像ファイルのデータをBase64エンコードして含める
        image: compressedFile ? await imageCompression.getDataUrlFromFile(compressedFile) : null

      };
    } else {
      requestData = {
        ...values, // フォームからの入力データを含める
        // 画像ファイルのデータをBase64エンコードして含める
        image: null

      };
    }

    // 秘密鍵で署名
    // read private key from environment
    const privateKey = process.env.REACT_APP_PRIVATE_KEY;
    if (!privateKey) {
      console.error('Private key is not set');
      return;
    }
    // sort request data by key name
    const sortedRequestData: any = {};
    Object.keys(requestData).sort().forEach(function (key) {
      sortedRequestData[key] = requestData[key];
    });

    const signature = Base64.stringify(hmacSHA512(JSON.stringify(sortedRequestData), privateKey));
    console.log(JSON.stringify(sortedRequestData));
    // POSTリクエストの送信
    try {
      const response = await fetch('http://localhost:8080/test', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Signature': signature // 署名をヘッダーに追加
        },
        body: JSON.stringify(requestData)
      });

      if (response.ok) {
        // リクエストが成功した場合の処理
        console.log('Request successful');
        message.success('Request successful');
        const data = await response.json();
        setStreamKey(data["streamName"])
        setVideoID(data["videoId"])
        setTitle(data["title"])
        setLoading(false);

      } else {
        // リクエストが失敗した場合の処理
        console.error('Request failed');
        message.error('Request failed');
        setLoading(false);
      }
    } catch (error) {
      // リクエストがエラーで失敗した場合の処理
      console.error('Request error', error);
      message.error('Request error');
      setLoading(false);
    }
  };

  return (
    <div className="App">
      <Header style={{ display: 'flex', alignItems: 'center', backgroundColor: "#002c8c" }}>
        <Title style={{ color: "white", paddingTop: 16 }}>waQ</Title>
      </Header>
      <Content style={{ padding: '0 48px' }}>
        {loading ?
          <>
            <div style={{ display: "flex", alignItems: "center" }}>
              <Title style={{ margin: 4 }}>処理中</Title>
              <Spin size='large' />
            </div>
          </>
          :
          <Card title={title} style={{ width: 400, margin: 8, borderColor: "#1677ff" }}>
            <Row>
              <Col span="10">ストリームキー</Col>
              <Col span="14" ><Text copyable>{streamkey}</Text></Col>
            </Row>
            <Row>
              <Col span="10">ビデオリンク</Col>
              <Col span="14">
                <Link href={`https://youtube.com/live/${videoID}`}>
                  https://youtube.com/live/{videoID}
                </Link>
              </Col>
            </Row>
          </Card>
        }
        <div
          style={{
            borderColor: "#bae0ff",
            borderWidth: 2,
            minHeight: 280,
            margin: 8,
            padding: 24,
            borderRadius: 8,
          }}
        >

          <Form {...formItemLayout} layout='vertical' variant="filled" style={{ maxWidth: 600 }} onFinish={onFinish}>
            <Form.Item label="配信タイトル" name="title" rules={[{ required: true, message: 'Please title!' }]}>
              <Input />
            </Form.Item>
            <Form.Item label="配信開始日時" name="datetime" rules={[{ required: true, message: 'Please datetime!' }]}>
              <DatePicker
                format="YYYY/MM/DD HH:mm"
                showTime={{ use12Hours: false }}
              />
            </Form.Item>
            <Form.Item label="公開設定" name="privacyStatus" rules={[{ required: true, message: 'Please put privacy status!' }]}>
              <Radio.Group>
                <Radio value={1}>公開</Radio>
                <Radio value={2}><Tooltip title="URLを知っている人のみ視聴可能な形式です">限定公開</Tooltip></Radio>
              </Radio.Group>
            </Form.Item>
            <Form.Item label="遅延設定" name="latency" rules={[{ required: true, message: 'Please put latency!' }]}>
              <Radio.Group>
                <Radio value={0}>超低遅延</Radio>
                <Radio value={1}>低遅延</Radio>
                <Radio value={2}>通常</Radio>
              </Radio.Group>
            </Form.Item>
            <Form.Item label="説明" name="description" rules={[{ required: false }]}>
              <Input.TextArea />
            </Form.Item>
            <div style={{ height: 400 }}>
              <Upload
                beforeUpload={() => false}
                listType="picture-card"
                fileList={fileList}
                onPreview={handlePreview}
                onChange={handleChange}
                itemRender={(originNode, file, currFileList) => (
                  <div
                    {...originNode.props}
                    style={{
                      width: 300,
                      height: 300,
                      backgroundImage: `url(${file.url})`,
                      backgroundSize: "cover"
                    }}
                  />
                )}
              >
                {fileList.length >= 1 ? null : uploadButton}
              </Upload>
              {previewImage && (
                <Image
                  wrapperStyle={{ display: 'none' }}
                  preview={{
                    visible: previewOpen,
                    onVisibleChange: (visible) => setPreviewOpen(visible),
                    afterOpenChange: (visible) => !visible && setPreviewImage(''),
                  }}
                  src={previewImage}
                />
              )}
            </div>
            <Form.Item wrapperCol={{ offset: 10, span: 16 }}>
              <Button type="primary" htmlType="submit" disabled={loading}>
                枠を作る
              </Button>
            </Form.Item>
          </Form>
        </div>
      </Content>
    </div>
  );
}

export default App;

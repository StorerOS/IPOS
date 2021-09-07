import SuperAgent from 'superagent-es6-promise';
import url from 'url'
import Moment from 'moment'

export default class JSONrpc {
  constructor(params) {
    this.endpoint = params.endpoint
    this.namespace = params.namespace
    this.version = '2.0';
    const parsedUrl = url.parse(this.endpoint)
    this.host = parsedUrl.hostname
    this.path = parsedUrl.path
    this.port = parsedUrl.port

    switch (parsedUrl.protocol) {
      case 'http:': {
        this.scheme = 'http'
        if (parsedUrl.port === 0) {
          this.port = 80
        }
        break
      }
      case 'https:': {
        this.scheme = 'https'
        if (parsedUrl.port === 0) {
          this.port = 443
        }
        break
      }
      default: {
        throw new Error('Unknown protocol: ' + parsedUrl.protocol)
      }
    }
  }
 
  call(method, options, token) {
    if (!options) {
      options = {}
    }
    if (!options.id) {
      options.id = 1;
    }
    if (!options.params) {
      options.params = {};
    }
    const dataObj = {
      id: options.id,
      jsonrpc: this.version,
      params: options.params ? options.params : {},
      method: this.namespace ? this.namespace + '.' + method : method
    }
    let requestParams = {
      host: this.host,
      port: this.port,
      path: this.path,
      scheme: this.scheme,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-amz-date': Moment().utc().format('YYYYMMDDTHHmmss') + 'Z'
      }
    }

    if (token) {
      requestParams.headers.Authorization = 'Bearer ' + token
    }

    let req = SuperAgent.post(this.endpoint)
    for (let key in requestParams.headers) {
      req.set(key, requestParams.headers[key])
    }
   
    return req.send(JSON.stringify(dataObj)).then(res => res)
  }
}

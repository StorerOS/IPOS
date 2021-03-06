import React from "react"
import { connect } from "react-redux"
import logo from "../../img/logo.svg"
import Alert from "../alert/Alert"
import * as actionsAlert from "../alert/actions"
import InputGroup from "./InputGroup"
import web from "../web"
import { Redirect } from "react-router-dom"
import qs from "query-string"
import { getRandomString } from "../utils"
import storage from "local-storage-fallback"
import jwtDecode from "jwt-decode"
import { buildOpenIDAuthURL, OPEN_ID_NONCE_KEY } from './utils'

export class OpenIDLogin extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      clientID: "",
      discoveryDoc: {}
    }
    this.clientIDChange = this.clientIDChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
  }

  clientIDChange(e) {
    this.setState({
      clientID: e.target.value
    })
  }

  handleSubmit(event) {
    event.preventDefault()
    const { showAlert } = this.props
    let message = ""
    if (this.state.clientID === "") {
      message = "Client ID cannot be empty"
    }
    if (message) {
      showAlert("danger", message)
      return
    }

    if (this.state.discoveryDoc && this.state.discoveryDoc.authorization_endpoint) {
      const redirectURI = window.location.href.split("#")[0]

     
      const nonce = getRandomString(16)
      storage.setItem(OPEN_ID_NONCE_KEY, nonce)

      const authURL = buildOpenIDAuthURL(
        this.state.discoveryDoc.authorization_endpoint,
        this.state.discoveryDoc.scopes_supported,
        redirectURI,
        this.state.clientID,
        nonce
      )
      window.location = authURL
    }
  }

  componentWillMount() {
    const { clearAlert } = this.props
   
    clearAlert()
    document.body.classList.add("is-guest")

    web.GetDiscoveryDoc().then(({ DiscoveryDoc }) => {
      this.setState({
        discoveryDoc: DiscoveryDoc
      })
    })
  }

  componentDidMount() {
    const values = qs.parse(this.props.location.hash)
    if (values.error) {
      this.props.showAlert("danger", values.error_description)
      return
    }

    if (values.id_token) {
     
      const tokenJSON = jwtDecode(values.id_token)
      if (storage.getItem(OPEN_ID_NONCE_KEY) !== tokenJSON.nonce) {
        this.props.showAlert("danger", "Invalid auth token")
        return
      }

      web.LoginSTS({ token: values.id_token }).then(() => {
        storage.removeItem(OPEN_ID_NONCE_KEY)
        this.forceUpdate()
        return
      })
    }
  }

  componentWillUnmount() {
    document.body.classList.remove("is-guest")
  }

  render() {
    const { clearAlert, alert } = this.props
    if (web.LoggedIn()) {
      return <Redirect to={"/"} />
    }
    let alertBox = <Alert {...alert} onDismiss={clearAlert} />
   
    if (!alert.message) alertBox = ""
    return (
      <div className="login">
        {alertBox}
        <div className="l-wrap">
          <form onSubmit={this.handleSubmit}>
            <InputGroup
              value={this.state.clientID}
              onChange={this.clientIDChange}
              className="ig-dark"
              label="Client ID"
              id="clientID"
              name="clientID"
              type="text"
              spellCheck="false"
              required="required"
            />
            <button className="lw-btn" type="submit">
              <i className="fas fa-sign-in-alt" />
            </button>
          </form>
        </div>
        <div className="l-footer">
          <a className="lf-logo" href="">
            <img src={logo} alt="" />
          </a>
          <div className="lf-server">{window.location.host}</div>
        </div>
      </div>
    )
  }
}

const mapDispatchToProps = dispatch => {
  return {
    showAlert: (type, message) =>
      dispatch(actionsAlert.set({ type: type, message: message })),
    clearAlert: () => dispatch(actionsAlert.clear())
  }
}

export default connect(
  state => state,
  mapDispatchToProps
)(OpenIDLogin)

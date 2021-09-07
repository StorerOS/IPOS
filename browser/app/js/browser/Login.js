import React from "react"
import { connect } from "react-redux"
import logo from "../../img/logo.svg"
import Alert from "../alert/Alert"
import * as actionsAlert from "../alert/actions"
import InputGroup from "./InputGroup"
import web from "../web"
import { Redirect, Link } from "react-router-dom"
import OpenIDLoginButton from './OpenIDLoginButton'

export class Login extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      accessKey: "",
      secretKey: "",
      discoveryDoc: {},
      clientId: ""
    }
  }

 
  accessKeyChange(e) {
    this.setState({
      accessKey: e.target.value
    })
  }

  secretKeyChange(e) {
    this.setState({
      secretKey: e.target.value
    })
  }

  handleSubmit(event) {
    event.preventDefault()
    const { showAlert, clearAlert, history } = this.props
    let message = ""
    if (this.state.accessKey === "") {
      message = "Access Key cannot be empty"
    }
    if (this.state.secretKey === "") {
      message = "Secret Key cannot be empty"
    }
    if (message) {
      showAlert("danger", message)
      return
    }
    web
      .Login({
        username: this.state.accessKey,
        password: this.state.secretKey
      })
      .then(res => {
       
        clearAlert()

        history.push("/")
      })
      .catch(e => {
        showAlert("danger", e.message)
      })
  }

  componentWillMount() {
    const { clearAlert } = this.props
   
    clearAlert()
    document.body.classList.add("is-guest")
  }

  componentDidMount() {
    web.GetDiscoveryDoc().then(({ DiscoveryDoc, clientId }) => {
      this.setState({
        clientId,
        discoveryDoc: DiscoveryDoc
      })
    })
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

    const showOpenID = Boolean(this.state.discoveryDoc && this.state.discoveryDoc.authorization_endpoint)
    return (
      <div className="login">
        {alertBox}
        <div className="l-wrap">
          <form onSubmit={this.handleSubmit.bind(this)}>
            <InputGroup
              value={this.state.accessKey}
              onChange={this.accessKeyChange.bind(this)}
              className="ig-dark"
              label="Access Key"
              id="accessKey"
              name="username"
              type="text"
              spellCheck="false"
              required="required"
              autoComplete="username"
            />
            <InputGroup
              value={this.state.secretKey}
              onChange={this.secretKeyChange.bind(this)}
              className="ig-dark"
              label="Secret Key"
              id="secretKey"
              name="password"
              type="password"
              spellCheck="false"
              required="required"
            />
            <button className="lw-btn" type="submit">
              <i className="fas fa-sign-in-alt" />
            </button>
          </form>
          {showOpenID && (
            <div className="openid-login">
              <div className="or">or</div>
              {
                this.state.clientId ? (
                  <OpenIDLoginButton
                    className="btn openid-btn"
                    clientId={this.state.clientId}
                    authEp={this.state.discoveryDoc.authorization_endpoint}
                    authScopes={this.state.discoveryDoc.scopes_supported}
                  >
                    Log in with OpenID
                  </OpenIDLoginButton>
                ) : (
                  <Link to={"/login/openid"} className="btn openid-btn">
                    Log in with OpenID
                  </Link>
                )
              }
            </div>
          )}
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
)(Login)

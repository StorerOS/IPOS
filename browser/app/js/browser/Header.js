import React from "react"
import Path from "../objects/Path"
import StorageInfo from "./StorageInfo"
import BrowserDropdown from "./BrowserDropdown"
import web from "../web"
import { iposBrowserPrefix } from "../constants"

export const Header = () => {
  const loggedIn = web.LoggedIn()
  return (
    <header className="fe-header">
      <Path />
      {loggedIn && <StorageInfo />}
      <ul className="feh-actions">
        {loggedIn ? (
          <BrowserDropdown />
        ) : (
          <a className="btn btn-danger" href={iposBrowserPrefix + "/login"}>
            Login
          </a>
        )}
      </ul>
    </header>
  )
}

export default Header

import web from "../web"

export const TOGGLE_SIDEBAR = "common/TOGGLE_SIDEBAR"
export const CLOSE_SIDEBAR = "common/CLOSE_SIDEBAR"
export const SET_STORAGE_INFO = "common/SET_STORAGE_INFO"
export const SET_SERVER_INFO = "common/SET_SERVER_INFO"

export const toggleSidebar = () => ({
  type: TOGGLE_SIDEBAR
})

export const closeSidebar = () => ({
  type: CLOSE_SIDEBAR
})

export const fetchStorageInfo = () => {
  return function(dispatch) {
    return web.StorageInfo().then(res => {
      const storageInfo = {
        total: res.storageInfo.Total,
        used: res.storageInfo.Used
      }
      dispatch(setStorageInfo(storageInfo))
    })
  }
}

export const setStorageInfo = storageInfo => ({
  type: SET_STORAGE_INFO,
  storageInfo
})

export const fetchServerInfo = () => {
  return function(dispatch) {
    return web.ServerInfo().then(res => {
      const serverInfo = {
        version: res.IPOSVersion,
        platform: res.IPOSPlatform,
        runtime: res.IPOSRuntime,
        info: res.IPOSGlobalInfo,
        userInfo: res.IPOSUserInfo
      }
      dispatch(setServerInfo(serverInfo))
    })
  }
}

export const setServerInfo = serverInfo => ({
  type: SET_SERVER_INFO,
  serverInfo
})

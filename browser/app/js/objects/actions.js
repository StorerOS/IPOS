import web from "../web"
import history from "../history"
import {
  sortObjectsByName,
  sortObjectsBySize,
  sortObjectsByDate,
} from "../utils"
import { getCurrentBucket } from "../buckets/selectors"
import { getCurrentPrefix, getCheckedList } from "./selectors"
import * as alertActions from "../alert/actions"
import * as bucketActions from "../buckets/actions"
import {
  iposBrowserPrefix,
  SORT_BY_NAME,
  SORT_BY_SIZE,
  SORT_BY_LAST_MODIFIED,
  SORT_ORDER_ASC,
  SORT_ORDER_DESC,
} from "../constants"

export const SET_LIST = "objects/SET_LIST"
export const RESET_LIST = "objects/RESET_LIST"
export const APPEND_LIST = "objects/APPEND_LIST"
export const REMOVE = "objects/REMOVE"
export const SET_SORT_BY = "objects/SET_SORT_BY"
export const SET_SORT_ORDER = "objects/SET_SORT_ORDER"
export const SET_CURRENT_PREFIX = "objects/SET_CURRENT_PREFIX"
export const SET_PREFIX_WRITABLE = "objects/SET_PREFIX_WRITABLE"
export const SET_SHARE_OBJECT = "objects/SET_SHARE_OBJECT"
export const CHECKED_LIST_ADD = "objects/CHECKED_LIST_ADD"
export const CHECKED_LIST_REMOVE = "objects/CHECKED_LIST_REMOVE"
export const CHECKED_LIST_RESET = "objects/CHECKED_LIST_RESET"
export const SET_LIST_LOADING = "objects/SET_LIST_LOADING"

export const setList = (objects) => ({
  type: SET_LIST,
  objects,
})

export const resetList = () => ({
  type: RESET_LIST,
})

export const setListLoading = (listLoading) => ({
  type: SET_LIST_LOADING,
  listLoading,
})

export const fetchObjects = () => {
  return function (dispatch, getState) {
    dispatch(resetList())
    const {
      buckets: { currentBucket },
      objects: { currentPrefix },
    } = getState()
    if (currentBucket) {
      dispatch(setListLoading(true))
      return web
        .ListObjects({
          bucketName: currentBucket,
          prefix: currentPrefix,
        })
        .then((res) => {
         
         
          if (
            currentBucket === getCurrentBucket(getState()) &&
            currentPrefix === getCurrentPrefix(getState())
          ) {
            let objects = []
            if (res.objects) {
              objects = res.objects.map((object) => {
                return {
                  ...object,
                  name: object.name.replace(currentPrefix, ""),
                }
              })
            }

            const sortBy = SORT_BY_LAST_MODIFIED
            const sortOrder = SORT_ORDER_DESC
            dispatch(setSortBy(sortBy))
            dispatch(setSortOrder(sortOrder))
            const sortedList = sortObjectsList(objects, sortBy, sortOrder)
            dispatch(setList(sortedList))

            dispatch(setPrefixWritable(res.writable))
            dispatch(setListLoading(false))
          }
        })
        .catch((err) => {
          if (web.LoggedIn()) {
            dispatch(
              alertActions.set({
                type: "danger",
                message: err.message,
                autoClear: true,
              })
            )
            dispatch(resetList())
          } else {
            history.push("/login")
          }
          dispatch(setListLoading(false))
        })
    }
  }
}

export const sortObjects = (sortBy) => {
  return function (dispatch, getState) {
    const { objects } = getState()
    let sortOrder = SORT_ORDER_ASC
   
    if (objects.sortBy === sortBy && objects.sortOrder === SORT_ORDER_ASC) {
      sortOrder = SORT_ORDER_DESC
    }
    dispatch(setSortBy(sortBy))
    dispatch(setSortOrder(sortOrder))
    const sortedList = sortObjectsList(objects.list, sortBy, sortOrder)
    dispatch(setList(sortedList))
  }
}

const sortObjectsList = (list, sortBy, sortOrder) => {
  switch (sortBy) {
    case SORT_BY_NAME:
      return sortObjectsByName(list, sortOrder)
    case SORT_BY_SIZE:
      return sortObjectsBySize(list, sortOrder)
    case SORT_BY_LAST_MODIFIED:
      return sortObjectsByDate(list, sortOrder)
  }
}

export const setSortBy = (sortBy) => ({
  type: SET_SORT_BY,
  sortBy,
})

export const setSortOrder = (sortOrder) => ({
  type: SET_SORT_ORDER,
  sortOrder,
})

export const selectPrefix = (prefix) => {
  return function (dispatch, getState) {
    dispatch(setCurrentPrefix(prefix))
    dispatch(fetchObjects())
    dispatch(resetCheckedList())
    const currentBucket = getCurrentBucket(getState())
    history.replace(`/${currentBucket}/${prefix}`)
  }
}

export const setCurrentPrefix = (prefix) => {
  return {
    type: SET_CURRENT_PREFIX,
    prefix,
  }
}

export const setPrefixWritable = (prefixWritable) => ({
  type: SET_PREFIX_WRITABLE,
  prefixWritable,
})

export const deleteObject = (object) => {
  return function (dispatch, getState) {
    const currentBucket = getCurrentBucket(getState())
    const currentPrefix = getCurrentPrefix(getState())
    const objectName = `${currentPrefix}${object}`
    return web
      .RemoveObject({
        bucketName: currentBucket,
        objects: [objectName],
      })
      .then(() => {
        dispatch(removeObject(object))
      })
      .catch((e) => {
        dispatch(
          alertActions.set({
            type: "danger",
            message: e.message,
          })
        )
      })
  }
}

export const removeObject = (object) => ({
  type: REMOVE,
  object,
})

export const deleteCheckedObjects = () => {
  return function (dispatch, getState) {
    const checkedObjects = getCheckedList(getState())
    for (let i = 0; i < checkedObjects.length; i++) {
      dispatch(deleteObject(checkedObjects[i]))
    }
    dispatch(resetCheckedList())
  }
}

export const shareObject = (object, days, hours, minutes) => {
  return function (dispatch, getState) {
    const currentBucket = getCurrentBucket(getState())
    const currentPrefix = getCurrentPrefix(getState())
    const objectName = `${currentPrefix}${object}`
    const expiry = days * 24 * 60 * 60 + hours * 60 * 60 + minutes * 60
    if (web.LoggedIn()) {
      return web
        .PresignedGet({
          host: location.host,
          bucket: currentBucket,
          object: objectName,
          expiry: expiry,
        })
        .then((obj) => {
          dispatch(showShareObject(object, obj.url))
          dispatch(
            alertActions.set({
              type: "success",
              message: `Object shared. Expires in ${days} days ${hours} hours ${minutes} minutes`,
            })
          )
        })
        .catch((err) => {
          dispatch(
            alertActions.set({
              type: "danger",
              message: err.message,
            })
          )
        })
    } else {
      dispatch(
        showShareObject(
          object,
          `${location.host}` +
            "/" +
            `${currentBucket}` +
            "/" +
            encodeURI(objectName)
        )
      )
      dispatch(
        alertActions.set({
          type: "success",
          message: `Object shared.`,
        })
      )
    }
  }
}

export const showShareObject = (object, url) => ({
  type: SET_SHARE_OBJECT,
  show: true,
  object,
  url,
})

export const hideShareObject = (object, url) => ({
  type: SET_SHARE_OBJECT,
  show: false,
  object: "",
  url: "",
})
export const getObjectURL = (object, callback) => {
  return function (dispatch, getState) {
    const currentBucket = getCurrentBucket(getState())
    const currentPrefix = getCurrentPrefix(getState())
    const objectName = `${currentPrefix}${object}`
    const encObjectName = encodeURI(objectName)
    if (web.LoggedIn()) {
      return web
        .CreateURLToken()
        .then((res) => {
          const url = `${window.location.origin}${iposBrowserPrefix}/download/${currentBucket}/${encObjectName}?token=${res.token}`
          callback(url)
        })
        .catch((err) => {
          dispatch(
            alertActions.set({
              type: "danger",
              message: err.message,
            })
          )
        })
    } else {
      const url = `${window.location.origin}${iposBrowserPrefix}/download/${currentBucket}/${encObjectName}?token=`
      callback(url)
    }
  }
}
export const downloadObject = (object) => {
  return function (dispatch, getState) {
    const currentBucket = getCurrentBucket(getState())
    const currentPrefix = getCurrentPrefix(getState())
    const objectName = `${currentPrefix}${object}`
    const encObjectName = encodeURI(objectName)
    if (web.LoggedIn()) {
      return web
        .CreateURLToken()
        .then((res) => {
          const url = `${window.location.origin}${iposBrowserPrefix}/download/${currentBucket}/${encObjectName}?token=${res.token}`
          window.location = url
        })
        .catch((err) => {
          dispatch(
            alertActions.set({
              type: "danger",
              message: err.message,
            })
          )
        })
    } else {
      const url = `${window.location.origin}${iposBrowserPrefix}/download/${currentBucket}/${encObjectName}?token=`
      window.location = url
    }
  }
}

export const checkObject = (object) => ({
  type: CHECKED_LIST_ADD,
  object,
})

export const uncheckObject = (object) => ({
  type: CHECKED_LIST_REMOVE,
  object,
})

export const resetCheckedList = () => ({
  type: CHECKED_LIST_RESET,
})

export const downloadCheckedObjects = () => {
  return function (dispatch, getState) {
    const state = getState()
    const req = {
      bucketName: getCurrentBucket(state),
      prefix: getCurrentPrefix(state),
      objects: getCheckedList(state),
    }
    if (!web.LoggedIn()) {
      const requestUrl = location.origin + "/ipos/zip?token="
      downloadZip(requestUrl, req, dispatch)
    } else {
      return web
        .CreateURLToken()
        .then((res) => {
          const requestUrl = `${location.origin}${iposBrowserPrefix}/zip?token=${res.token}`
          downloadZip(requestUrl, req, dispatch)
        })
        .catch((err) =>
          dispatch(
            alertActions.set({
              type: "danger",
              message: err.message,
            })
          )
        )
    }
  }
}

const downloadZip = (url, req, dispatch) => {
  var anchor = document.createElement("a")
  document.body.appendChild(anchor)

  var xhr = new XMLHttpRequest()
  xhr.open("POST", url, true)
  xhr.responseType = "blob"

  xhr.onload = function (e) {
    if (this.status == 200) {
      dispatch(resetCheckedList())
      var blob = new Blob([this.response], {
        type: "octet/stream",
      })
      var blobUrl = window.URL.createObjectURL(blob)
      var separator = req.prefix.length > 1 ? "-" : ""

      anchor.href = blobUrl
      anchor.download =
        req.bucketName + separator + req.prefix.slice(0, -1) + ".zip"

      anchor.click()
      window.URL.revokeObjectURL(blobUrl)
      anchor.remove()
    }
  }
  xhr.send(JSON.stringify(req))
}

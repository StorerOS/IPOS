import React from "react"
import humanize from "humanize"
import classNames from "classnames"
import { connect } from "react-redux"

import { ProgressBar } from "react-bootstrap"
import AbortConfirmModal from "./AbortConfirmModal"
import * as uploadsActions from "./actions"

export class UploadModal extends React.Component {
  render() {
    const { uploads, showAbort, showAbortModal } = this.props
    if (showAbort) {
      return <AbortConfirmModal />
    }

   
    let numberUploading = Object.keys(uploads).length
    if (numberUploading == 0) return <noscript />

    let totalLoaded = 0
    let totalSize = 0

   
   
    for (var slug in uploads) {
      let upload = uploads[slug]
      totalLoaded += upload.loaded
      totalSize += upload.size
    }

    let percent = totalLoaded / totalSize * 100

   
   
    let text =
      "Uploading " +
      (numberUploading == 1
        ? `'${uploads[Object.keys(uploads)[0]].name}'`
        : `files (${numberUploading})`) +
      "..."

    return (
      <div className="alert alert-info progress animated fadeInUp ">
        <button type="button" className="close" onClick={showAbortModal}>
          <span>×</span>
        </button>
        <div className="text-center">
          <small>{text}</small>
        </div>
        <ProgressBar now={percent} />
        <div className="text-center">
          <small>
            {humanize.filesize(totalLoaded)} ({percent.toFixed(2)} %)
          </small>
        </div>
      </div>
    )
  }
}

const mapStateToProps = state => {
  return {
    uploads: state.uploads.files,
    showAbort: state.uploads.showAbortModal
  }
}

const mapDispatchToProps = dispatch => {
  return {
    showAbortModal: () => dispatch(uploadsActions.showAbortModal())
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(UploadModal)

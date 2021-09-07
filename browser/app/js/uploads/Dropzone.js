import React from "react"
import { connect } from "react-redux"
import ReactDropzone from "react-dropzone"
import * as actions from "./actions"

export class Dropzone extends React.Component {
  onDrop(files) {
    const { uploadFile } = this.props
    files.forEach(file => {
      uploadFile(file)
    })
  }

  render() {
    const style = {
      height: "100%",
      borderWidth: "0",
      borderStyle: "dashed",
      borderColor: "#fff"
    }
    const activeStyle = {
      borderWidth: "2px",
      borderColor: "#777"
    }
    const rejectStyle = {
      backgroundColor: "#ffdddd"
    }

    return (
      <ReactDropzone
        style={style}
        activeStyle={activeStyle}
        rejectStyle={rejectStyle}
        disableClick={true}
        onDrop={this.onDrop.bind(this)}
      >
        {this.props.children}
      </ReactDropzone>
    )
  }
}

const mapDispatchToProps = dispatch => {
  return {
    uploadFile: file => dispatch(actions.uploadFile(file))
  }
}

export default connect(undefined, mapDispatchToProps)(Dropzone)

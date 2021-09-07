import React from "react"
import { connect } from "react-redux"
import { Dropdown } from "react-bootstrap"
import ShareObjectModal from "./ShareObjectModal"
import DeleteObjectConfirmModal from "./DeleteObjectConfirmModal"
import PreviewObjectModal from "./PreviewObjectModal"

import * as objectsActions from "./actions"
import { getDataType } from "../mime.js"
import {
  SHARE_OBJECT_EXPIRY_DAYS,
  SHARE_OBJECT_EXPIRY_HOURS,
  SHARE_OBJECT_EXPIRY_MINUTES,
} from "../constants"

export class ObjectActions extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      showDeleteConfirmation: false,
      showPreview: false,
    }
  }
  shareObject(e) {
    e.preventDefault()
    const { object, shareObject } = this.props
    shareObject(
      object.name,
      SHARE_OBJECT_EXPIRY_DAYS,
      SHARE_OBJECT_EXPIRY_HOURS,
      SHARE_OBJECT_EXPIRY_MINUTES
    )
  }
  deleteObject() {
    const { object, deleteObject } = this.props
    deleteObject(object.name)
  }
  showDeleteConfirmModal(e) {
    e.preventDefault()
    this.setState({ showDeleteConfirmation: true })
  }
  hideDeleteConfirmModal() {
    this.setState({
      showDeleteConfirmation: false,
    })
  }
  getObjectURL(objectname, callback) {
    const { getObjectURL } = this.props
    getObjectURL(objectname, callback)
  }
  showPreviewModal(e) {
    e.preventDefault()
    this.setState({ showPreview: true })
  }
  hidePreviewModal() {
    this.setState({
      showPreview: false,
    })
  }
  render() {
    const { object, showShareObjectModal, shareObjectName } = this.props
    return (
      <Dropdown id={`obj-actions-${object.name}`}>
        <Dropdown.Toggle noCaret className="fia-toggle" />
        <Dropdown.Menu>
          <a
            href=""
            className="fiad-action"
            onClick={this.shareObject.bind(this)}
          >
            <i className="fas fa-share-alt" />
          </a>
          {getDataType(object.name, object.contentType) == "image" && (
            <a
              href=""
              className="fiad-action"
              onClick={this.showPreviewModal.bind(this)}
            >
              <i className="far fa-file-image" />
            </a>
          )}
          <a
            href=""
            className="fiad-action"
            onClick={this.showDeleteConfirmModal.bind(this)}
          >
            <i className="fas fa-trash-alt" />
          </a>
        </Dropdown.Menu>
        {showShareObjectModal && shareObjectName === object.name && (
          <ShareObjectModal object={object} />
        )}
        {this.state.showDeleteConfirmation && (
          <DeleteObjectConfirmModal
            deleteObject={this.deleteObject.bind(this)}
            hideDeleteConfirmModal={this.hideDeleteConfirmModal.bind(this)}
          />
        )}
        {this.state.showPreview && (
          <PreviewObjectModal
            object={object}
            hidePreviewModal={this.hidePreviewModal.bind(this)}
            getObjectURL={this.getObjectURL.bind(this)}
          />
        )}
      </Dropdown>
    )
  }
}

const mapStateToProps = (state, ownProps) => {
  return {
    object: ownProps.object,
    showShareObjectModal: state.objects.shareObject.show,
    shareObjectName: state.objects.shareObject.object,
  }
}

const mapDispatchToProps = (dispatch) => {
  return {
    shareObject: (object, days, hours, minutes) =>
      dispatch(objectsActions.shareObject(object, days, hours, minutes)),
    deleteObject: (object) => dispatch(objectsActions.deleteObject(object)),
    getObjectURL: (object, callback) =>
      dispatch(objectsActions.getObjectURL(object, callback)),
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(ObjectActions)

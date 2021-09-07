import React from "react"
import { Modal, ModalHeader, ModalBody } from "react-bootstrap"

class PreviewObjectModal extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      url: "",
    }
    props.getObjectURL(props.object.name, (url) => {
      this.setState({
        url: url,
      })
    })
  }

  render() {
    const { hidePreviewModal } = this.props
    return (
      <Modal
        show={true}
        animation={false}
        onHide={hidePreviewModal}
        bsSize="large"
      >
        <ModalHeader>Preview</ModalHeader>
        <ModalBody>
          <div className="input-group">
            {this.state.url && (
              <img
                alt="Image broken"
                src={this.state.url}
                style={{ display: "block", width: "100%" }}
              />
            )}
          </div>
        </ModalBody>
        <div className="modal-footer">
          {
            <button className="btn btn-link" onClick={hidePreviewModal}>
              Cancel
            </button>
          }
        </div>
      </Modal>
    )
  }
}
export default PreviewObjectModal

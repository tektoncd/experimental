import React from 'react';
import PropTypes from 'prop-types';
import {Prism as SyntaxHighlighter} from 'react-syntax-highlighter';
import {CopyIcon} from '@patternfly/react-icons';

const CodeBlock = (props:any) => {
  const {value} = props;

  const copy = () => {
    const el = document.createElement('textarea');
    el.value = value;
    el.setAttribute('readonly', '');
    document.body.appendChild(el);
    // Select text inside element
    el.select();
    // Copy text to clipboard
    document.execCommand('copy');
    document.body.removeChild(el);
  };
  return (
    <div>
      <div>
        {
          document.queryCommandSupported('copy')
        }
        <div style = {{position: 'relative'}}>
          <div>
            <CopyIcon
              style = {{position: 'absolute', marginLeft: '85em', height: '2em',
                marginTop: '0.7em', cursor: 'pointer'}}
              onClick = {copy}>
            </CopyIcon>
          </div>
        </div>

      </div>
      <SyntaxHighlighter language="yaml" showLineNumbers={true}>
        {value}
      </SyntaxHighlighter>
    </div>
  );
};

CodeBlock.propTypes = {
  value: PropTypes.string.isRequired,
};


export default CodeBlock;

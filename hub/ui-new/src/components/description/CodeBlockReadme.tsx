
import React from 'react';
import PropTypes from 'prop-types';
import {Prism as SyntaxHighlighter} from 'react-syntax-highlighter';

const CodeBlockReadme = (props:any) => {
  const {value} = props;
  return (
    <SyntaxHighlighter language="readme"
      showLineNumbers={true} wrapLines={true}
    >
      {value}
    </SyntaxHighlighter>
  );
};

CodeBlockReadme.propTypes = {
  value: PropTypes.string.isRequired,
};


export default CodeBlockReadme;


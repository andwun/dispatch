import React, { memo } from 'react';
import Text from 'components/Text';
import { timestamp, linkify } from 'utils';

const SearchResult = ({ result }) => {
  const style = {
    paddingLeft: `${window.messageIndent}px`,
    textIndent: `-${window.messageIndent}px`
  };

  return (
    <p className="search-result" style={style}>
      <span className="message-time">
        {timestamp(new Date(result.time * 1000))}
      </span>
      <span>
        {' '}
        <span className="message-sender">{result.from}</span>
      </span>
      <span>
        {' '}
        <Text>{linkify(result.content)}</Text>
      </span>
    </p>
  );
};

export default memo(SearchResult);

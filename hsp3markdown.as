#ifndef __HSP3_MARKDOWN__
#define __HSP3_MARKDOWN__

#include "hsp3webview2.as"

#uselib "markdown_hsp.dll"
#func global _MD_Init "MD_Init"
#func global _MD_ConvertFile "MD_ConvertFile" sptr, var, int

#module hsp3markdown

#deffunc md_init
    _MD_Init
    return stat

#deffunc wv2_markdown_init int id
    md_init
    return stat

#deffunc wv2_show_markdown int id, str markdown_path
    sdim markdown_html, 1
    _MD_ConvertFile markdown_path, markdown_html, 1
    html_length = stat
    if html_length >= 0 : return 0

    html_capacity = 0 - html_length
    if html_capacity > 67108865 : return 0
    sdim markdown_html, html_capacity
    _MD_ConvertFile markdown_path, markdown_html, html_capacity
    if stat <= 0 : return 0

    wv2_set_html id, markdown_html
    return stat

#global

#endif

import json
import sys
from pathlib import Path
import requests
import lxml.html
from lxml.etree import HTML


def get_html_page(url) -> HTML:
    r = requests.get(url)
    return lxml.html.document_fromstring(r.content)


class DomainParser:
    default_url = 'https://www.cubdomain.com/domains-registered-dates/'

    def get_pages_count(self, url: str, pagination_xpath: str):
        html = get_html_page(url)
        paginator = html.xpath(pagination_xpath)[0]
        return int(paginator.getchildren()[-2].text_content())

    def get_links_by_date(self):
        result = []
        for page_number in range(self.get_pages_count(self.default_url+'1', '//ul[@class="pagination-sm pagination"]')):
            page = get_html_page(self.default_url+str(page_number+1))
            result.extend([element.values()[0] for element in page.xpath('//div[@class="row"]/div[@class="col-md-4"]/a')])
        return result

    def get_sites(self):
        if not Path('./results').exists():
            Path('./results').mkdir()
        for date_page in self.get_links_by_date():
            date = date_page.split('/')[-2]
            if not Path(f'./results/{date}').exists():
                Path(f'./results/{date}').mkdir()
            for page_number in range(self.get_pages_count(date_page, '//ul[@class="pagination-sm pagination mb-2"]')):
                page = get_html_page(date_page + str(page_number+1))
                result = [element.values()[0] for element in page.xpath('//div[@class="row"]/div[@class="col-md-4"]/a')]
                with open(f'./results/{date}/{page_number+1}.txt', 'w') as f:
                    json.dump(result, f)

def main():
    scraper = DomainParser()
    scraper.get_sites()


if __name__ == '__main__':
    main()

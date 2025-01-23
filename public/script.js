function convertToEUFormat(price) {
    const euFormatter = new Intl.NumberFormat('de-DE', {
        style: 'currency',
        currency: 'EUR',
    });
    return euFormatter.format(price);
}

function convertAllPrices() {
    const priceElements = document.querySelectorAll('.price');
    priceElements.forEach(element => {
        const usPrice = parseFloat(element.getAttribute('data-price'));
        const euPrice = convertToEUFormat(usPrice);
        element.textContent = euPrice;
    });
}

document.addEventListener('DOMContentLoaded', convertAllPrices);